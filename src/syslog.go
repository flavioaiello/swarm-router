package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	bufferSize = 65536
	socketPath = "/dev/log"
)

func getSeverity(code int) string {
	switch code & 0x07 {
	case 0:
		return "emerg"
	case 1:
		return "alert"
	case 2:
		return "crit"
	case 3:
		return "err"
	case 4:
		return "warning"
	case 5:
		return "notice"
	case 6:
		return "info"
	case 7:
		return "debug"
	default:
		return "unknown"
	}
}

func readData(data []byte) {
	header := "n.a."
	message := string(data)
	endOfCode := strings.Index(message, ">")
	if endOfCode != -1 && endOfCode < 5 {
		code, err := strconv.Atoi(string(data[1:endOfCode]))
		if err == nil {
			header = getSeverity(code)
		}
		message = string(data[endOfCode+1:])
	}
	log.Printf("%s %s", header, message)
}

func syslog() {
	if _, err := os.Stat(socketPath); nil == err {
		os.Remove(socketPath)
	}
	unixAddr, err := net.ResolveUnixAddr("unixgram", socketPath)
	if err != nil {
		log.Fatalf("Resolv error: %s", err.Error())
	}
	conn, err := net.ListenUnixgram("unixgram", unixAddr)
	if err != nil {
		log.Fatalf("Listen error: %s", err.Error())
	}
	if err := os.Chmod(socketPath, 0777); nil != err {
		log.Fatalf("Socket permission error: %s", err.Error())
	}
	reader := bufio.NewReader(conn)
	for {
		buffer := make([]byte, bufferSize)
		size, err := reader.Read(buffer)
		if err != nil {
			log.Fatalf("Syslog read error: %s", err.Error())
		}
		go readData(buffer[0:size])
	}
}