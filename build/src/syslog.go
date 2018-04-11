package main

import (
  "net"
  "bufio"
  "log"
  "strconv"
  "strings"
  "os"
)

const (
    bufferSize = 65536
    socketPath = "/dev/log"
)

type syslog struct {}

func (syslog syslog) getFacility(code int) string {
    switch code >> 3 {
        case 0: return "kern"
        case 1: return "user"
        case 2: return "mail"
        case 3: return "daemon"
        case 4: return "auth"
        case 5: return "syslog"
        case 6: return "lpr"
        case 7: return "news"
        case 8: return "uucp"
        case 9: return "cron"
        case 10: return "authpriv"
        case 11: return "ftp"
        case 12: return "ntp"
        case 13: return "security"
        case 14: return "console"
        case 15: return "mark"
        case 16: return "local0"
        case 17: return "local1"
        case 18: return "local2"
        case 19: return "local3"
        case 20: return "local4"
        case 21: return "local5"
        case 22: return "local6"
        case 23: return "local7"
        default: return "unknown"
    }
}

func (syslog syslog) getSeverity(code int) string {
    switch code & 0x07 {
        case 0: return "emerg"
        case 1: return "alert"
        case 2: return "crit"
        case 3: return "err"
        case 4: return "warning"
        case 5: return "notice"
        case 6: return "info"
        case 7: return "debug"
        default: return "unknown"
    }
}

func (syslog syslog) listen(connection net.Conn) {
    reader := bufio.NewReader(connection)
    for {
        buffer := make([]byte, bufferSize)
        size, err := reader.Read(buffer)
        if err != nil {
            log.Fatal("Syslog read error: %s", err.Error())
        }
        go syslog.readData(buffer[0:size])
    }
}

func (syslog syslog) readData(data []byte) {
    header := "unknown:unknown"
    message := string(data)
    endOfCode := strings.Index(message, ">")
    if -1 != endOfCode && 5 > endOfCode {
        code, err := strconv.Atoi(string(data[1:endOfCode]))
        if nil == err {
            header = syslog.getFacility(code) + ":" + syslog.getSeverity(code)
        }
        message = string(data[endOfCode + 1:])
    }
    log.Printf("%s: %s\n", header, strings.TrimSuffix(message, "\n"))
}

func (syslog syslog) run() {
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
    syslog.listen(conn)
}
