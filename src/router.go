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
	backends = struct {
		sync.RWMutex
		active    bool
		endpoints map[string]bool
	}{endpoints: make(map[string]bool)}
)

func reload() {
	<-throttle
	if !backends.active {
		// generate configuration
		log.Printf("generate haproxy configuration")
		executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")
		// reload haproxy
		log.Printf("reload haproxy SIGUSR2 PID %d", pid)
		syscall.Kill(pid, syscall.SIGUSR2)
		// set status
		log.Printf("backends activated")
		backends.Lock()
		backends.active = true
		backends.Unlock()
	}
}

func doneChan(done chan int) {
	done <- 1
}

func router(done chan int, port string) {
	defer doneChan(done)
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
		upstream, err := net.Dial("tcp", getBackendHostname(hostname)+":"+getBackendPort(hostname, false))
		if err != nil {
			log.Printf("Backend connection error: %s", err.Error())
			downstream.Close()
			return
		}
		addBackend(hostname, false)
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

func getIP(hostname string) net.IP {
	// Resolve target ip address for hostname
	backendIPAddr, err := net.ResolveIPAddr("ip", hostname)
	if err != nil {
		log.Printf("Error resolving target ip address: %s", err.Error())
	}
	return backendIPAddr.IP
}

func getBackendHostname(hostname string) string {
	if (fqdnBackendsHostname != "true") {
		return strings.Split(hostname, ".")[0]
	}
	return hostname
}

func getBackendPort(hostname string, encryption bool) string {
	var backendPort string
	if encryption {
		// Set default tls port
		backendPort = tlsBackendsDefaultPort
		// Set special port if any
		for _, portOverride := range strings.Split(tlsBackendsPort, " ") {
			if portOverride != "" {
				backend, port, _ := net.SplitHostPort(portOverride)
				if strings.HasPrefix(hostname, backend) {
					backendPort = port
					break
				}
			}
		}
	} else {
		// Set default http port
		backendPort = httpBackendsDefaultPort
		// Set special port if any
		for _, portOverride := range strings.Split(httpBackendsPort, " ") {
			if portOverride != "" {
				backend, port, _ := net.SplitHostPort(portOverride)
				if strings.HasPrefix(hostname, backend) {
					backendPort = port
					break
				}
			}
		}
	}
	return backendPort
}

func isMember(hostname string) bool {
	// Resolve target ip address for hostname
	backendIP := getIP(hostname)
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

func addBackend(hostname string, encryption bool) {
	defer cleanupBackends()
	if _, exists := backends.endpoints[hostname]; !exists {
		backends.Lock()
		defer backends.Unlock()
		backends.endpoints[hostname] = encryption
		backends.active = false
		log.Printf("Adding %s to swarm-router", hostname)
		go reload()
	}
}

func delBackend(hostname string) {
	defer cleanupBackends()
	if _, exists := backends.endpoints[hostname]; exists {
		backends.Lock()
		defer backends.Unlock()
		delete(backends.endpoints, hostname)
		backends.active = false
		log.Printf("Removing %s from swarm-router", hostname)
		go reload()
	}
}

func cleanupBackends() {
	for hostname := range backends.endpoints {
		if !isMember(hostname) {
			backends.Lock()
			defer backends.Unlock()
			delete(backends.endpoints, hostname)
			backends.active = false
			log.Printf("Removing %s from swarm-router due to cleanup", hostname)
			go reload()
		}
	}
}