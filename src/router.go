package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	throttle = time.Tick(7 * time.Second)
	routes   = struct {
		sync.RWMutex
		active   bool
		mappings map[string]string
	}{mappings: make(map[string]string)}
)

func reload() {
	<-throttle
	if !routes.active {
		// generate configuration
		log.Printf("generate haproxy configuration")
		executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")
		// reload haproxy
		log.Printf("reload haproxy SIGUSR2 PID %d", pid)
		syscall.Kill(pid, syscall.SIGUSR2)
		// set status
		log.Printf("routes activated")
		routes.Lock()
		routes.active = true
		routes.Unlock()
	}
}

func router(exit chan bool, port string) {
	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		log.Printf("Listening error: %s", err.Error())
		return
	}
	log.Printf("Listening started on port: %s", port)
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %s", err.Error())
			return
		}
		go handle(connection)
	}
	exit <- true
}

func handle(downstream net.Conn) {
	var hostname string
	var read []byte
	reader := bufio.NewReader(downstream)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			log.Printf("Error reading: %s", err.Error())
			downstream.Close()
			return
		}
		read = append(read, line...)
		read = append(read, "\n"...)
		if bytes.HasPrefix(line, []byte("Host: ")) {
			hostname = string(bytes.TrimPrefix(line, []byte("Host: ")))
			if strings.ContainsAny(hostname, ":") {
				hostname, _, err = net.SplitHostPort(hostname)
				if err != nil {
					log.Printf("Error splitting hostname: %s", err.Error())
					downstream.Close()
					return
				}
			}
			break
		}
	}
	if isMember(hostname) {
		backend := getBackend(hostname)
		upstream, err := net.Dial("tcp", backend)
		if err != nil {
			log.Printf("Backend connection error: %s", err.Error())
			downstream.Close()
			return
		}
		addRoute(hostname, backend)
		log.Printf("Transient proxying: %s", hostname)
		go func() {
			upstream.Write(read)
			io.Copy(upstream, reader)
			upstream.Close()
		}()
		go func() {
			io.Copy(downstream, upstream)
			log.Printf("Closing transient downstream")
			downstream.Close()
		}()
	} else {
		downstream.Close()
	}
}

func isMember(hostname string) bool {
	// Resolve target ip address for hostname
	backendIP := getBackendIP(hostname)
	// Get own ip adresses
	ownIPAddrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("Error resolving own ip addresses: %s", err.Error())
		return false
	}
	// Check if target ip is member of attached swarm networks
	for _, ownIPAddr := range ownIPAddrs {
		if ownIPNet, state := ownIPAddr.(*net.IPNet); state && !ownIPNet.IP.IsLoopback() && ownIPNet.IP.To4() != nil {
			if ownIPNet.Contains(backendIP) {
				return true
			}
		}
	}
	return false
}

func addRoute(hostname, backend string) {
	defer cleanupRoutes()
	if _, exists := routes.mappings[hostname]; !exists {
		routes.Lock()
		defer routes.Unlock()
		routes.mappings[hostname] = backend
		routes.active = false
		log.Printf("Adding %s to swarm-router", hostname)
		go reload()
	}
}

func delRoute(hostname string) {
	defer cleanupRoutes()
	if _, exists := routes.mappings[hostname]; exists {
		routes.Lock()
		defer routes.Unlock()
		delete(routes.mappings, hostname)
		routes.active = false
		log.Printf("Removing %s from swarm-router", hostname)
		go reload()
	}
}

func cleanupRoutes() {
	for hostname := range routes.mappings {
		if !isMember(hostname) {
			routes.Lock()
			defer routes.Unlock()
			delete(routes.mappings, hostname)
			routes.active = false
			log.Printf("Removing %s from swarm-router due to cleanup", hostname)
			go reload()
		}
	}
}

func getBackendIP(hostname string) net.IP {
	// Resolve target ip address for hostname
	backendIPAddr, err := net.ResolveIPAddr("ip", hostname)
	if err != nil {
		log.Printf("Error resolving target ip address: %s", err.Error())
	}
	return backendIPAddr.IP
}

func getBackend(hostname string) string {
	var backend string
	fqdn := strings.Split(hostname, ".")
	for i := range fqdn {
		// check fqdn for service shortnames
		hostname = strings.Join(fqdn[0:i+1], ".")
		log.Printf("searchBackend hostname: %s", hostname)
		// Search default port for fqdn
		for _, searchPort := range strings.Split(defaultBackendPorts, " ") {
			if searchPort != "" {
				upstream, _ := net.Dial("tcp", net.JoinHostPort(hostname, searchPort))
				if upstream != nil {
					upstream.Close()
					backend = net.JoinHostPort(hostname, searchPort)
					return backend
				}
			}
		}
		// Set special port if any
		for _, portOverride := range strings.Split(overrideBackendPorts, " ") {
			if portOverride != "" {
				backend, port, _ := net.SplitHostPort(portOverride)
				if strings.HasPrefix(hostname, backend) {
					backend = net.JoinHostPort(hostname, port)
					return backend
				}
			}
		}
	}
	return backend
}
