package main

import (
	"log"
	"net"
	"strings"
	"sync"
	"time"
        "syscall"
        "net/http"
        "net/http/httputil"
)

var (
	throttle = time.Tick(7 * time.Second)
	reloads = 0
	backends = struct {
		sync.RWMutex
		active    bool
		endpoints map[string]bool
	}{endpoints: make(map[string]bool)}
)

func router() {

        http.HandleFunc("/", proxyHandler)
        log.Fatal(http.ListenAndServe(":" + httpSwarmRouterPort, nil))
}

func reload() {
	<-throttle
	if !backends.active || reloads == 0 {

	        log.Printf("recreate haproxy configuration")
		executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")

        	log.Printf("reload haproxy: send SIGUSR2 to PID %d", pid)
	        syscall.Kill(pid, syscall.SIGUSR2)

        	log.Printf("set backend status")
		backends.active = true

        	log.Printf("increment reloads counter at %d", reloads)
		reloads++
	}
}

func newDirector(r *http.Request) func(*http.Request) {
        return func(req *http.Request) {
                req.URL.Scheme = "http"
                req.URL.Host = r.Host + ":" + getBackendPort(r.Host, false)
		if isMember(r.Host) {
			addBackend(r.Host, false)
		}
                reqLog, err := httputil.DumpRequestOut(req, false)
                if err != nil {
                        log.Printf("Got error %s\n %+v\n", err.Error(), req)
                }
                log.Println(string(reqLog))
        }
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
        proxy := &httputil.ReverseProxy{
                Transport: &http.Transport{},
                Director:  newDirector(r),
        }
        proxy.ServeHTTP(w, r)
}

func addBackend(hostname string, encryption bool) {
	defer cleanupBackends()
	if _, exists := backends.endpoints[hostname]; !exists {
		backends.Lock()
		defer backends.Unlock()
		backends.endpoints[hostname] = encryption
		backends.active = false
		log.Printf("Adding %s to swarm-router", hostname)
		reload()
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
		reload()
	}
}

func cleanupBackends() {
	for hostname := range backends.endpoints {
		if !isMember(getBackendHostname(hostname)) {
			backends.Lock()
			defer backends.Unlock()
			delete(backends.endpoints, hostname)
			backends.active = false
			log.Printf("Removing %s from swarm-router due to cleanup", hostname)
			reload()
		}
	}
}

func getBackendPort(hostname string, encryption bool) string {
	backendPort := "0"
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

func getBackendHostname(hostname string) string {
	if strings.ContainsAny(hostname, ":") {
		hostname, _, _ = net.SplitHostPort(hostname)
	}
	if dnsBackendSuffix != "" {
		hostname = hostname + dnsBackendSuffix
	}
	return hostname
}

func isMember(hostname string) bool {
	// Resolve target ip address for hostname
	backendIPAddr, err := net.ResolveIPAddr("ip", getBackendHostname(hostname))
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
				//log.Printf("Target ip address %s for %s is part of swarm network %s", backendIPAddr.String(), getBackendHostname(hostname), ownIPNet)
				return true
			}
			//log.Printf("Target ip address %s for %s is not part of swarm network %s", backendIPAddr.String(), getBackendHostname(hostname), ownIPNet)
		}
	}
	return false
}
