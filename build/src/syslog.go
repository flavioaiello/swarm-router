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

func listen(connection net.Conn) {
	reader := bufio.NewReader(connection)
	for {
		buffer := make([]byte, bufferSize)
		size, err := reader.Read(buffer)
		if err != nil {
			log.Fatal("Syslog read error: %s", err.Error())
		}
		go readData(buffer[0:size])
	}
}

func readData(data []byte) {
  header := "null:null"
	message := string(data)
	endOfCode := strings.Index(message, ">")
	if -1 != endOfCode && 5 > endOfCode {
		code, err := strconv.Atoi(string(data[1:endOfCode]))
		if nil == err {
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
	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{socketPath, "unixgram"})
	if nil != err {
		log.Fatal("Listen error: %s", err.Error())
	}
	if err := os.Chmod(socketPath, 0777); nil != err {
		log.Fatal("Socket permission error: %s", err.Error())
	}
	listen(conn)
}
