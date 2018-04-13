package main

import (
	"os"
	"strconv"
	"strings"
)

// listener ports
var httpPorts = getEnv("HTTP_PORTS", "80")
var tlsPorts = getEnv("TLS_PORTS", "443")

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
var dnsBackendFqdn, err = strconv.ParseBool(getEnv("DNS_BACKEND_FQDN", "true"))

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
		os.Setenv(key, defaultValue)
	}
	return value
}

func main() {

	// Init haproxy config
	executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")

	// Start syslog
	go syslog()

	// Start haproxy
	go haproxy()

	// Start swarm-router config listeners
	httpDone := make(chan int)
	go defaultBackend(httpDone, 10080, httpHandler)
	tlsDone := make(chan int)
	go defaultBackend(tlsDone, 10443, tlsHandler)
	<-httpDone
	<-tlsDone
}
