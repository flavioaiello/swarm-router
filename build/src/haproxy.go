package main

import (
  "bytes"
  "io"
  "bufio"
  "log"
  "net"
  "strings"
  "strconv"
  "os"
  "os/exec"
  "container/list"
  "time"
  "reflect"
  "sync"
)

var rate = time.Second
var throttle = time.Tick(7 * rate)

// temp backend maps
var (
  tempHttpBackends = make(map[string]int)
  tempHttpBackendsLock sync.RWMutex
)
var (
  tempTlsBackends = make(map[string]int)
  tempTlsBackendsLock sync.RWMutex
)

func run(program string, args ...string) string {
  var stdout, stderr bytes.Buffer
  cmd := exec.Command(program, args...)
  cmd.Stdin = os.Stdin;
  cmd.Stdout = &stdout;
  cmd.Stderr = &stderr;
  err := cmd.Run()
  if err != nil {
      log.Printf("Error: %s", err.Error())
      log.Printf("Stderr: %s", stderr.String())
      os.Exit(1)
  }
  return stdout.String()
}

func haproxy() {
  _ = run("haproxy","-db","-f","/usr/local/etc/haproxy/haproxy.cfg")
}

func getPids() string {
  return strings.TrimSpace(run("pidof","haproxy"))
}

func reload(){
  <-throttle
  if !reflect.DeepEqual(tempHttpBackends, httpBackends){
    tempHttpBackendsLock.Lock()
    for key, value := range httpBackends {
      tempHttpBackends[key] = value
    }
    tempHttpBackendsLock.Unlock()
    // Generate new haproxy configuration
    executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")
    // reload haproxy
    log.Printf("******************* reload haproxy ******************** ")
    run("haproxy","-db","-f","/usr/local/etc/haproxy/haproxy.cfg","-x","/run/haproxy.sock","-sf",getPids())
  }
}

func defaultBackend(done chan int, port int, handle func(net.Conn)) {
  defer doneChan(done)
  listener, err := net.Listen("tcp", "127.0.0.1:" + httpSwarmRouterPort)
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

func doneChan(done chan int){
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

func addBackend(hostname string){
  // Add new backend to backend memory map (ttl map pending)
  log.Printf("Adding %s to swarm-router", hostname)
  httpBackendsLock.Lock()
  httpBackends[hostname] = getBackendPort(hostname)
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
    bytes, _, error := reader.ReadLine()
    if error != nil {
      log.Printf("Error reading", error)
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
  // Resolve target ip address for hostname
  backendIPAddr, err := net.ResolveIPAddr("ip", hostname)
  if err != nil {
      log.Printf("Error resolving ip address for: %s", err.Error())
      downstream.Close()
      return
  }
  // Get swarm-router ip adresses
  ownIPAddrs, err := net.InterfaceAddrs()
  if err != nil {
      log.Printf("Error resolving own ip address: %s", err.Error())
      downstream.Close()
      return
  }
  for _, ownIPAddr := range ownIPAddrs {
    if ownIPNet, state := ownIPAddr.(*net.IPNet); state && !ownIPNet.IP.IsLoopback() && ownIPNet.IP.To4() != nil {
      // Check if target ip is member of attached swarm networks
      if ownIPNet.Contains(backendIPAddr.IP) {
        upstream, err := net.Dial("tcp", getBackend(hostname) + ":" + strconv.Itoa(getBackendPort(hostname)))
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
      }
      //log.Printf("Target ip address %s for %s is not part of swarm network %s", backendIPAddr.String(), getBackend(hostname), ownIPNet)
    }
  }
}
