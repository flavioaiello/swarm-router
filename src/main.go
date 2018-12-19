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
var httpSwarmRouterPort, _ = strconv.Atoi(getEnv("HTTP_SWARM_ROUTER_PORT", "10080"))
var tlsSwarmRouterPort, _ = strconv.Atoi(getEnv("TLS_SWARM_ROUTER_PORT", "10443"))

// backends default ports
var httpBackendsDefaultPort, _ = strconv.Atoi(getEnv("HTTP_BACKENDS_DEFAULT_PORT", "8080"))
var tlsBackendsDefaultPort, _ = strconv.Atoi(getEnv("TLS_BACKENDS_DEFAULT_PORT", "8443"))

// backends port rules
var httpBackendsPort = strings.Split(getEnv("HTTP_BACKENDS_PORT", ""), " ")
var tlsBackendsPort = strings.Split(getEnv("TLS_BACKENDS_PORT", ""), " ")

// backend dns modes
var dnsBackendSuffix = getEnv("DNS_BACKEND_SUFFIX", "")
var dnsBackendFqdn, _ = strconv.ParseBool(getEnv("DNS_BACKEND_FQDN", "true"))

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
		os.Setenv(key, defaultValue)
	}
	return strings.TrimSpace(value)
}

func main() {

	// Start syslog
	go syslog()

	// Reload haproxy
	reload()
	
	// Start swarm-router
	httpDone := make(chan int)
	swarmRouter(httpDone, httpSwarmRouterPort, httpHandler)
	<-httpDone
}
