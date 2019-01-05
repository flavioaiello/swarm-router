package main

import (
	"os"
	"os/exec"
	"log"
	"strings"
)

var (
	// haproxy master pid
	pid = 0

	// listener ports
	httpPorts = getEnv("HTTP_PORTS", "80")
	tlsPorts = getEnv("TLS_PORTS", "443")

	// swarm router ports
	httpSwarmRouterPort = getEnv("HTTP_SWARM_ROUTER_PORT", "10080")
	tlsSwarmRouterPort = getEnv("TLS_SWARM_ROUTER_PORT", "10443")

	// backends default ports
	httpBackendsDefaultPort = getEnv("HTTP_BACKENDS_DEFAULT_PORT", "8080")
	tlsBackendsDefaultPort = getEnv("TLS_BACKENDS_DEFAULT_PORT", "8443")

	// backends port rules
	httpBackendsPort = getEnv("HTTP_BACKENDS_PORT", "")
	tlsBackendsPort = getEnv("TLS_BACKENDS_PORT", "")

	// backend dns suffix
	dnsBackendSuffix = getEnv("DNS_BACKEND_SUFFIX", "")
)

func getEnv(key, defaultValue string) string {
	// get env vars eg. set if not present
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
		os.Setenv(key, defaultValue)
	}
	return strings.TrimSpace(value)
}

func init() {
	// haproxy config
	executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")
}

func main() {
	// start router
	go router()

	// start haproxy 
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("Start error: %s", err.Error())
	}
	pid = cmd.Process.Pid
	log.Printf("Started haproxy master process with pid: %d", pid)
	err := cmd.Wait()
	log.Printf("Exit error: %s", err.Error())
}