package main

import (
	"github.com/robfig/cron"
	"log"
	"os"
	"os/exec"
	"strings"
)

var (
	// haproxy master pid
	pid = 0

	// swarm listeners
	httpPorts = getEnv("HTTP_PORTS", "80 8080")
	tlsPorts  = getEnv("TLS_PORTS", "443 8443")

	// swarm router port
	swarmRouterPort = getEnv("SWARM_ROUTER_PORT", "35353")

	// backends default ports
	defaultBackendPorts = getEnv("DEFAULT_BACKEND_PORTS", "80 443 8000 8080 8443 9000")

	// backends port rules
	overrideBackendPorts = getEnv("OVERRIDE_BACKEND_PORTS", "")

	// backends verify tls
	backendsVerifyTLS = getEnv("BACKENDS_VERIFY_TLS", "")
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

func haproxy(exit chan bool) {
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
	exit <- true
}

func init() {
	// haproxy config
	executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")
}

func main() {
	// Cleanup scheduler
	c := cron.New()
	c.AddFunc("@daily", reload)
	c.Start()
	// Start swarm-router and haproxy
	exit := make(chan bool, 1)
	go router(exit, swarmRouterPort)
	go haproxy(exit)
	<-exit
}
