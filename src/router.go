package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"syscall"
)

func reload() {
	// Cleanup backends
	for backend := range backendMap() {
		if !isMember(backend) {
			os.Unsetenv(backend)
		}
	}
	// Generate configuration
	log.Printf("Generate haproxy configuration")
	executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")
	// Reload haproxy
	log.Printf("Reload haproxy SIGUSR2 PID %d", pid)
	syscall.Kill(pid, syscall.SIGUSR2)
	// Print actual number of go routines
	log.Printf("Running go routines: %d", runtime.NumGoroutine())
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

func handle(srcConn net.Conn) {
	var hostname string
	var read []byte
	defer srcConn.Close()
	reader := bufio.NewReader(srcConn)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			log.Printf("Error reading: %s", err.Error())
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
					return
				}
			}
			log.Printf("Hostname found: %s", hostname)
			break
		}
	}
	if isMember(hostname) {
		backend := getBackend(hostname)
		dstConn, err := net.Dial("tcp", backend)
		if err != nil {
			log.Printf("Backend connection error: %s", err.Error())
			return
		}
		defer dstConn.Close()
		go func() {
			os.Setenv("BE_"+hostname, backend)
			reload()
		}()
		errc := make(chan error, 1)
		go func(chan<- error) {
			dstConn.Write(read)
			_, err := io.Copy(dstConn, reader)
			log.Printf("Transient proxying: %s", hostname)
			errc <- err
		}(errc)
		go func(chan<- error) {
			_, err := io.Copy(srcConn, dstConn)
			log.Printf("Closing transient proxy")
			errc <- err
		}(errc)
		<-errc
	}
}

func isMember(hostname string) bool {
	// Resolve target ip address for hostname
	log.Printf("Resolve target ip address for hostname: %s", hostname)
	backendIPAddr, err := net.ResolveIPAddr("ip", hostname)
	if err != nil {
		log.Printf("Error resolving target ip address: %s", err.Error())
		return false
	}
	// Get own ip adresses
	ownIPAddrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("Error resolving own ip addresses: %s", err.Error())
		return false
	}
	// Check if target ip is member of attached swarm networks
	for _, ownIPAddr := range ownIPAddrs {
		if ownIPNet, state := ownIPAddr.(*net.IPNet); state && !ownIPNet.IP.IsLoopback() && ownIPNet.IP.To4() != nil {
			if ownIPNet.Contains(backendIPAddr.IP) {
				return true
			}
		}
	}
	return false
}

func getBackend(hostname string) string {
	var backend string
	log.Printf("searchBackend hostname: %s", hostname)
	// Search default port for fqdn
	for _, searchPort := range strings.Split(defaultBackendPorts, " ") {
		if searchPort != "" {
			dstConn, _ := net.Dial("tcp", net.JoinHostPort(hostname, searchPort))
			if dstConn != nil {
				dstConn.Close()
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
	log.Printf("Backend found: %s", backend)
	return backend
}
