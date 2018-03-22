package main

import (
  "os"
  "strings"
)

// swarm router ports
var httpSwarmRouterPort = getEnv("HTTP_SWARM_ROUTER_PORT", "10080")
var tlsSwarmRouterPort = getEnv("TLS_SWARM_ROUTER_PORT", "10443")

// backends default ports
var httpBackendsDefaultPort = getEnv("HTTP_BACKENDS_DEFAULT_PORT", "8080")
var tlsBackendsDefaultPort = getEnv("TLS_BACKENDS_DEFAULT_PORT", "8443")

// backends port rules
var httpBackendsPort = strings.Split(getEnv("HTTP_BACKENDS_PORT", ""), " ")
var tlsBackendsPort = strings.Split(getEnv("TLS_BACKENDS_PORT", ""), " ")

// backend dns modes
var dnsBackendSuffix = getEnv("DNS_BACKEND_SUFFIX", "")
var dnsBackendFqdn = getEnv("DNS_BACKEND_FQDN", "true")

// Backend maps
var httpBackends = make(map[string]int)
var tlsBackends = make(map[string]int)

func main() {

  // Init haproxy config
  executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")

  // Start syslog socket
	syslog := Syslog{}
  go syslog.run()

  // Start haproxy
  go haproxy()

  // Start proxy
	httpDone := make(chan int)
	go defaultBackend(httpDone, 10080, httpHandler)
	//tlsDone := make(chan int)
	//go defaultBackend(tlsDone, 10443, tlsHandler)
	<-httpDone
	//<-tlsDone
}

func getEnv(key, defaultValue string) string {
    value, exists := os.LookupEnv(key)
    if !exists {
        value = defaultValue
    }
    return value
}
