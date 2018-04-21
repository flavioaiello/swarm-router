package main

import (
	"bufio"
	"bytes"
	"container/list"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	rate     = time.Second
	throttle = time.Tick(7 * rate)
)

var backends = struct {
	sync.RWMutex
	active    bool
	endpoints map[string]bool
}{endpoints: make(map[string]bool)}

func addBackend(endpoint string, encryption bool) {
	defer backends.Unlock()
	defer cleanupBackends()
	if _, exists := backends.endpoints[endpoint]; !exists {
		backends.Lock()
		backends.endpoints[endpoint] = encryption
		backends.active = false
		log.Printf("Adding %s to swarm-router", endpoint)
		reload()
	}
}

func delBackend(endpoint string) {
	defer backends.Unlock()
	defer cleanupBackends()
	if _, exists := backends.endpoints[endpoint]; exists {
		backends.Lock()
		delete(backends.endpoints, endpoint)
		backends.active = false
		log.Printf("Removing %s from swarm-router", endpoint)
		reload()
	}
}

func cleanupBackends() {
	defer backends.Unlock()
	for endpoint, _ := range backends.endpoints {
		if !isMember(getBackendHostname(endpoint)) {
			backends.Lock()
			delete(backends.endpoints, endpoint)
			backends.active = false
			log.Printf("Removing %s from swarm-router due to cleanup", endpoint)
			reload()
		}
	}
}

func getBackendPort(endpoint string, encryption bool) int {
	backendPort := 0
	if !encryption {
		// Set default http port
		backendPort = httpBackendsDefaultPort
		// Set special port if any
		for i := range httpBackendsPort {
			backend, port, _ := net.SplitHostPort(httpBackendsPort[i])
			if strings.HasPrefix(endpoint, backend) {
				backendPort, _ = strconv.Atoi(port)
				break
			}
		}
	} else {
		// Set default tls port
		backendPort = tlsBackendsDefaultPort
		// Set special port if any
		for i := range httpBackendsPort {
			backend, port, _ := net.SplitHostPort(tlsBackendsPort[i])
			if strings.HasPrefix(endpoint, backend) {
				backendPort, _ = strconv.Atoi(port)
				break
			}
		}
	}
	return backendPort
}

func getBackendHostname(endpoint string) string {
	hostname = endpoint
	if strings.ContainsAny(hostname, ":") {
	  hostname, _, _ := net.SplitHostPort(hostname)
	}
	if !dnsBackendFqdn {
		hostname = strings.Split(hostname, ".")[0]
	}
	if dnsBackendSuffix != "" {
		hostname = hostname + dnsBackendSuffix
	}
	return hostname
}

func isMember(endpoint string) bool {
	// Resolve target ip address for hostname
	backendIPAddr, err := net.ResolveIPAddr("ip", getBackendHostname(endpoint))
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
				//log.Printf("Target ip address %s for %s is part of swarm network %s", backendIPAddr.String(), getBackend(hostname), ownIPNet)
				return true
			}
			//log.Printf("Target ip address %s for %s is not part of swarm network %s", backendIPAddr.String(), getBackend(hostname), ownIPNet)
		}
	}
	return false
}

func runCommand(program string, args ...string) string {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(program, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Error: %s\nStderr: %s", err.Error(), stderr.String())
	}
	return stdout.String()
}

func haproxy() {
	_ = runCommand("haproxy", "-db", "-f", "/usr/local/etc/haproxy/haproxy.cfg")
}

func getPids() string {
	return strings.TrimSpace(runCommand("pidof", "haproxy", "-s"))
}

func reload() {
	<-throttle
	if !backends.active {
		// Generate new haproxy configuration
		executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")
		// Reload haproxy
		log.Printf("*** Reload haproxy ***")
		runCommand("haproxy", "-db", "-f", "/usr/local/etc/haproxy/haproxy.cfg", "-x", "/run/haproxy.sock", "-sf", getPids())
		// Set backends status
		backends.active = true
	}
}

func swarmRouter(done chan int, port int, handle func(net.Conn)) {
	defer doneChan(done)
	listener, err := net.Listen("tcp", "127.0.0.1:" + strconv.Itoa(port))
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
	endpoint := ""
	readLines := list.New()
	for endpoint == "" {
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
			endpoint = strings.TrimPrefix(line, "Host: ")
			break
		}
	}
	log.Printf("Endpoint: %s", endpoint)
	log.Printf("Hostname: %s", getBackendHostname(endpoint))
	log.Printf("Port: %d", getBackendPort(endpoint, false))
	if isMember(endpoint) {
		upstream, err := net.Dial("tcp", getBackendHostname(endpoint)+":"+strconv.Itoa(getBackendPort(endpoint, false)))
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
		go addBackend(endpoint, false)
	} else {
		downstream.Close()
	}
}

func tlsHandler(downstream net.Conn) {
	firstByte := make([]byte, 1)
	_, err := downstream.Read(firstByte)
	if err != nil {
		log.Printf("Could not read first byte: %s", err.Error())
		return
	} else if firstByte[0] != 0x16 {
		log.Printf("No TLS protocol error: %s", err.Error())
		return
	}
	versionBytes := make([]byte, 2)
	_, err = downstream.Read(versionBytes)
	if err != nil {
		log.Printf("Could not read TLS version: %s", err.Error())
		return
	} else if versionBytes[0] < 3 || (versionBytes[0] == 3 && versionBytes[1] < 1) {
		log.Printf("Expecting TLS - aborting now  ...")
		return
	}
	restLengthBytes := make([]byte, 2)
	_, err = downstream.Read(restLengthBytes)
	if err != nil {
		log.Printf("Could not read rest length: %s", err.Error())
		return
	}
	restLength := (int(restLengthBytes[0]) << 8) + int(restLengthBytes[1])
	rest := make([]byte, restLength)
	_, err = downstream.Read(rest)
	if err != nil {
		log.Printf("Could not read rest data: %s", err.Error())
		return
	}
	handshakeType := rest[0]
	if handshakeType != 0x1 {
		log.Printf("Not client hello")
		return
	}
	// Start and skip fixed lenght headers
	current := 38
	// Skip over session ID
	sessionIDLength := int(rest[current])
	current += 1 + sessionIDLength
	cipherSuiteLength := (int(rest[current]) << 8) + int(rest[current+1])
	current += 2 + cipherSuiteLength
	compressionMethodLength := int(rest[current])
	current += 1 + compressionMethodLength
	if current > restLength {
		log.Printf("No extensions found")
		return
	}
	current += 2
	endpoint := ""
	for current < restLength && endpoint == "" {
		extensionType := (int(rest[current]) << 8) + int(rest[current+1])
		current += 2
		extensionDataLength := (int(rest[current]) << 8) + int(rest[current+1])
		current += 2
		if extensionType == 0 {
			// Skip over number of names as we're assuming there's just one
			current += 2
			nameType := rest[current]
			current += 1
			if nameType != 0 {
				log.Printf("Not a hostname")
				return
			}
			nameLen := (int(rest[current]) << 8) + int(rest[current+1])
			current += 2
			endpoint = string(rest[current : current+nameLen])
		}
		current += extensionDataLength
	}
	log.Printf("Endpoint: %s", endpoint)
	log.Printf("Hostname: %s", getBackendHostname(endpoint))
	log.Printf("Port: %d", getBackendPort(endpoint, false))
	if isMember(endpoint) {
		upstream, err := net.Dial("tcp", getBackendHostname(endpoint)+":"+strconv.Itoa(getBackendPort(endpoint, true)))
		if err != nil {
			log.Printf("Backend connection error: %s", err.Error())
			downstream.Close()
			return
		}
		upstream.Write(firstByte)
		upstream.Write(versionBytes)
		upstream.Write(restLengthBytes)
		upstream.Write(rest)
		go copy(upstream, downstream)
		go copy(downstream, upstream)
		go addBackend(endpoint, true)
	} else {
		downstream.Close()
	}
}
