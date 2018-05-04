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
	for requestHeader, _ := range backends.endpoints {
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
		if len(tlsBackendsPort) > 1 {
			for i := range tlsBackendsPort {
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
		if len(httpBackendsPort) > 1 {
			for i := range httpBackendsPort {
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

func tlsHandler(downstream net.Conn) {
	firstByte := make([]byte, 1)
	_, err := downstream.Read(firstByte)
	if err != nil {
		log.Printf("First byte error: %s", err.Error())
		return
	}
	if firstByte[0] != 0x16 {
		log.Printf("Not TLS error")
	}
	versionBytes := make([]byte, 2)
	_, err = downstream.Read(versionBytes)
	if err != nil {
		log.Printf("Version bytes error: %s", err.Error())
		return
	}
	if versionBytes[0] < 3 || (versionBytes[0] == 3 && versionBytes[1] < 1) {
		log.Printf("SSL not TLS error")
		return
	}
	restLengthBytes := make([]byte, 2)
	_, err = downstream.Read(restLengthBytes)
	if err != nil {
		log.Printf("restLength byte error: %s", err.Error())
		return
	}
	restLength := (int(restLengthBytes[0]) << 8) + int(restLengthBytes[1])

	rest := make([]byte, restLength)
	_, err = downstream.Read(rest)
	if err != nil {
		log.Printf("Rest bytes error: %s", err.Error())
		return
	}
	current := 0
	handshakeType := rest[0]
	current += 1
	if handshakeType != 0x1 {
		log.Printf("ClientHello missing error")
		return
	}
	// Skip over another length
	current += 3
	// Skip over protocolversion
	current += 2
	// Skip over random number
	current += 4 + 28
	// Skip over session ID
	sessionIDLength := int(rest[current])
	current += 1
	current += sessionIDLength
	cipherSuiteLength := (int(rest[current]) << 8) + int(rest[current+1])
	current += 2
	current += cipherSuiteLength
	compressionMethodLength := int(rest[current])
	current += 1
	current += compressionMethodLength
	if current > restLength {
		log.Printf("Extensions missing error")
		return
	}
	// Skip over extensionsLength
	// extensionsLength := (int(rest[current]) << 8) + int(rest[current + 1])
	current += 2
	sniHeader := ""
	for current < restLength && sniHeader == "" {
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
				log.Printf("SNI header missing error")
				return
			}
			nameLen := (int(rest[current]) << 8) + int(rest[current+1])
			current += 2
			sniHeader = string(rest[current : current+nameLen])
		}
		current += extensionDataLength
	}
	if isMember(sniHeader) {
		upstream, err := net.Dial("tcp", getBackendHostname(sniHeader)+":"+strconv.Itoa(getBackendPort(sniHeader, true)))
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
		go addBackend(sniHeader, true)
	} else {
		downstream.Close()
	}
}
