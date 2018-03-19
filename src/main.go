package main

import (

)

var backends = make(map[string]string)

func main() {

  // Init haproxy config
  executeTemplate("/usr/local/etc/haproxy/haproxy.tmpl", "/usr/local/etc/haproxy/haproxy.cfg")

  // Start syslog socket
	syslog := Syslog{}
  go syslog.run()

	// Start listner
	httpDone := make(chan int)
	go listner(httpDone, 10080, handler)
	<-httpDone
}
