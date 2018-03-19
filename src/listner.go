package main

import (
	"bufio"
	"log"
	"net"
	"strings"
	"strconv"
  "os/exec"
)

func listner(tchan chan int, port int, handle func(net.Conn)) {
	defer doneChan(tchan)
	listener, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(port))
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

func doneChan(tchan chan int){
	tchan <- 1
}

func handler(downstream net.Conn) {
	defer downstream.Close()
	hostname := ""
	scanner := bufio.NewScanner(downstream)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "Host: ") {
			hostname = strings.TrimPrefix(scanner.Text(), "Host: ")
			break
		}
	}
  // Check if backend was already added
	if backends[hostname] == "" {
    // Resolve target ip address for hostname
		backendIPAddr, err := net.ResolveIPAddr("ip", hostname)
		if err != nil {
				log.Printf("Error resolving ip address for: %s", err.Error())
				return
		}
		// Get swarm-router ip adresses
		ownIPAddrs, err := net.InterfaceAddrs()
		if err != nil {
        log.Printf("Error resolving own ip address: %s", err.Error())
				return
	  }
		for _, a := range ownIPAddrs {
			if ownIPNet, ok := a.(*net.IPNet); ok && !ownIPNet.IP.IsLoopback() {
				if ownIPNet.IP.To4() != nil {
					// Check if target ip is member of attached swarm networks
					if ownIPNet.Contains(backendIPAddr.IP) {
					  addBackend(hostname)
						break
					}
					log.Printf("Target ip address %s for %s is not part of swarm network", backendIPAddr.String(), hostname)
				}
			}
		}
	}
}

func addBackend(hostname string) {
	// Add new backend to backend memory map (ttl map pending)
	log.Printf("Adding %s to swarm-router", hostname)
	backends[hostname] = hostname
	// Generate new haproxy configuration
	executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")
	// Restart haproxy using USR2 signal
	_, err := exec.Command("/usr/bin/supervisorctl", "restart", "haproxy").CombinedOutput()
	if err != nil {
		log.Printf("Restart error: %s", err.Error())
		return
	}
}
