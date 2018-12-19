package main

import (
	"bufio"
	"container/list"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
        "syscall"
)

var (
	rate     = time.Second
	throttle = time.Tick(7 * rate)
)

var reloads = 0

var backends = struct {
	sync.RWMutex
	active    bool
	endpoints map[string]bool
}{endpoints: make(map[string]bool)}

func addBackend(requestHeader string, encryption bool) {
	defer cleanupBackends()
	if _, exists := backends.endpoints[requestHeader]; !exists {
		backends.Lock()
		defer backends.Unlock()
		backends.endpoints[requestHeader] = encryption
		backends.active = false
		log.Printf("Adding %s to swarm-router", requestHeader)
		go reload()
	}
}

func delBackend(requestHeader string) {
	defer cleanupBackends()
	if _, exists := backends.endpoints[requestHeader]; exists {
		backends.Lock()
		defer backends.Unlock()
		delete(backends.endpoints, requestHeader)
		backends.active = false
		log.Printf("Removing %s from swarm-router", requestHeader)
		go reload()
	}
}

func cleanupBackends() {
	for requestHeader := range backends.endpoints {
		if !isMember(getBackendHostname(requestHeader)) {
			backends.Lock()
			defer backends.Unlock()
			delete(backends.endpoints, requestHeader)
			backends.active = false
			log.Printf("Removing %s from swarm-router due to cleanup", requestHeader)
			go reload()
		}
	}
}

func getBackendPort(requestHeader string, encryption bool) int {
	backendPort := 0
	if encryption {
		// Set default tls port
		backendPort = tlsBackendsDefaultPort
		// Set special port if any
		for i := range tlsBackendsPort {
			if tlsBackendsPort[i] != "" {
				backend, port, _ := net.SplitHostPort(tlsBackendsPort[i])
				if strings.HasPrefix(requestHeader, backend) {
					backendPort, _ = strconv.Atoi(port)
					break
				}
			}
		}
	} else {
		// Set default http port
		backendPort = httpBackendsDefaultPort
		// Set special port if any
		for i := range httpBackendsPort {
			if httpBackendsPort[i] != "" {
				backend, port, _ := net.SplitHostPort(httpBackendsPort[i])
				if strings.HasPrefix(requestHeader, backend) {
					backendPort, _ = strconv.Atoi(port)
					break
				}
			}
		}
	}
	return backendPort
}

func getBackendHostname(requestHeader string) string {
	hostname := requestHeader
	if strings.ContainsAny(hostname, ":") {
		hostname, _, _ = net.SplitHostPort(hostname)
	}
	if !dnsBackendFqdn {
		hostname = strings.Split(hostname, ".")[0]
	}
	if dnsBackendSuffix != "" {
		hostname = hostname + dnsBackendSuffix
	}
	return hostname
}

func isMember(requestHeader string) bool {
	// Resolve target ip address for hostname
	backendIPAddr, err := net.ResolveIPAddr("ip", getBackendHostname(requestHeader))
	if err != nil {
		log.Printf("Error resolving ip address: %s", err.Error())
		return false
	}
	// Get swarm-router ip adresses
	ownIPAddrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("Error resolving own ip addresses: %s", err.Error())
		return false
	}
	for _, ownIPAddr := range ownIPAddrs {
		if ownIPNet, state := ownIPAddr.(*net.IPNet); state && !ownIPNet.IP.IsLoopback() && ownIPNet.IP.To4() != nil {
			// Check if target ip is member of attached swarm networks
			if ownIPNet.Contains(backendIPAddr.IP) {
				//log.Printf("Target ip address %s for %s is part of swarm network %s", backendIPAddr.String(), getBackendHostname(requestHeader), ownIPNet)
				return true
			}
			//log.Printf("Target ip address %s for %s is not part of swarm network %s", backendIPAddr.String(), getBackendHostname(requestHeader), ownIPNet)
		}
	}
	return false
}

func reload() {
	<-throttle
	if !backends.active || reloads == 0 {

	        log.Printf("*** Regenerate haproxy configuration ***")
		executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")

        	log.Printf("*** Reload %d haproxy ***", reloads)
	        syscall.Kill(1, syscall.SIGUSR2)

        	log.Printf("*** Set backend status and reload iterator ***")
		backends.active = true
		reloads++
	}
}


func swarmRouter(done chan int, port int, handle func(net.Conn)) {
	defer doneChan(done)
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		log.Printf("Listening error: %s", err.Error())
		return
	}
	log.Printf("Listening started on port: %d", port)
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %s", err.Error())
			return
		}
		go handle(connection)
	}
}


func doneChan(done chan int) {
	done <- 1
}


func copy(dst io.WriteCloser, src io.Reader) {
	io.Copy(dst, src)
	dst.Close()
}


func httpHandler(downstream net.Conn) {
	reader := bufio.NewReader(downstream)
	hostnameHeader := ""
	readLines := list.New()
	for hostnameHeader == "" {
		bytes, _, err := reader.ReadLine()
		if err != nil {
			log.Printf("Error reading: %s", err.Error())
			downstream.Close()
			return
		}
		line := string(bytes)
		readLines.PushBack(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Host: ") {
			hostnameHeader = strings.TrimPrefix(line, "Host: ")
			break
		}
	}
	if isMember(hostnameHeader) {
		upstream, err := net.Dial("tcp", getBackendHostname(hostnameHeader)+":"+strconv.Itoa(getBackendPort(hostnameHeader, false)))
		if err != nil {
			log.Printf("Backend connection error: %s", err.Error())
			downstream.Close()
			return
		}
		for element := readLines.Front(); element != nil; element = element.Next() {
			line := element.Value.(string)
			upstream.Write([]byte(line))
			upstream.Write([]byte("\n"))
		}
		go copy(upstream, reader)
		go copy(downstream, upstream)
		go addBackend(hostnameHeader, false)
	} else {
		downstream.Close()
	}
}
