package main

import (

)

var backends = make(map[string]int)

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
