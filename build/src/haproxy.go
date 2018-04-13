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
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var rate = time.Second
var throttle = time.Tick(7 * rate)

// Backend temp maps
var (
	tempHttpBackends     = make(map[string]int)
	tempHttpBackendsLock sync.RWMutex
)
var (
	tempTlsBackends     = make(map[string]int)
	tempTlsBackendsLock sync.RWMutex
)

// Backend maps
var (
	httpBackends     = make(map[string]int)
	httpBackendsLock sync.RWMutex
)
var (
	tlsBackends     = make(map[string]int)
	tlsBackendsLock sync.RWMutex
)

func run(program string, args ...string) string {
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
	_ = run("haproxy", "-db", "-f", "/usr/local/etc/haproxy/haproxy.cfg")
}

func getPids() string {
	return strings.TrimSpace(run("pidof", "haproxy", "-s"))
}

func reload() {
	<-throttle
	if !reflect.DeepEqual(tempHttpBackends, httpBackends) {
		tempHttpBackendsLock.Lock()
		for key, value := range httpBackends {
			tempHttpBackends[key] = value
		}
		tempHttpBackendsLock.Unlock()
		// Generate new haproxy configuration
		executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")
		// reload haproxy
		log.Printf("Reload haproxy")
		run("haproxy", "-db", "-f", "/usr/local/etc/haproxy/haproxy.cfg", "-x", "/run/haproxy.sock", "-sf", getPids())
	}
}

func defaultBackend(done chan int, port int, handle func(net.Conn)) {
	defer doneChan(done)
	listener, err := net.Listen("tcp", "127.0.0.1:"+httpSwarmRouterPort)
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

func getBackend(hostname string) string {
	if !dnsBackendFqdn {
		hostname = strings.Split(hostname, ".")[0]
	}
	if dnsBackendSuffix != "" {
		hostname = hostname + dnsBackendSuffix
	}
	return hostname
}

func getBackendPort(hostname string) int {
	backendPort := 0
	for i := range httpBackendsPort {
		backend, port, _ := net.SplitHostPort(httpBackendsPort[i])
		if strings.HasPrefix(hostname, backend) {
			backendPort, _ = strconv.Atoi(port)
			break
		}
		backendPort, _ = strconv.Atoi(httpBackendsDefaultPort)
	}
	return backendPort
}

func addBackend(hostname string) {
	// Add new backend to backend memory map (ttl map pending)
	log.Printf("Adding %s to swarm-router", hostname)
	httpBackendsLock.Lock()
	// Add backend to map
	httpBackends[hostname] = getBackendPort(hostname)
	// Cleanup backends
	for key, value := range httpBackends {
		if !isMemberOfSwarm(key) {
			log.Printf("Removing %s:%d from swarm-router", key, value)
			delete(httpBackends, key)
		}
	}
	httpBackendsLock.Unlock()
	go reload()
}

func copy(dst io.WriteCloser, src io.Reader) {
	io.Copy(dst, src)
	dst.Close()
}

func httpHandler(downstream net.Conn) {
	reader := bufio.NewReader(downstream)
	hostname := ""
	readLines := list.New()
	for hostname == "" {
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
			hostname = strings.TrimPrefix(line, "Host: ")
			if strings.ContainsAny(hostname, ":") {
				hostname, _, _ = net.SplitHostPort(hostname)
			}
			break
		}
	}
	if isMemberOfSwarm(getBackend(hostname)) {
		upstream, err := net.Dial("tcp", getBackend(hostname)+":"+strconv.Itoa(getBackendPort(hostname)))
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
		if httpBackends[hostname] == 0 {
			go addBackend(getBackend(hostname))
		}
	} else {
		downstream.Close()
	}
}

func isMemberOfSwarm(hostname string) bool {
	// Resolve target ip address for hostname
	backendIPAddr, err := net.ResolveIPAddr("ip", hostname)
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
