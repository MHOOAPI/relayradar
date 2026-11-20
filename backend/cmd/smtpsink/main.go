// Command smtpsink is a minimal SMTP server for local testing of the email
// alert channel. It accepts any message and logs the envelope plus body.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:2525", "listen address")
	flag.Parse()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("smtp sink on %s", *addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := func(s string) { fmt.Fprint(conn, s+"\r\n") }

	w("220 sink ESMTP")
	var from, to string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			w("250-sink")
			w("250 8BITMIME")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			from = strings.TrimSpace(line[len("MAIL FROM:"):])
			w("250 OK")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			to = strings.TrimSpace(line[len("RCPT TO:"):])
			w("250 OK")
		case cmd == "DATA":
			w("354 End data with <CR><LF>.<CR><LF>")
			var body strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(l, "\r\n") == "." {
					break
				}
				body.WriteString(l)
			}
			log.Printf("mail from=%s to=%s\n%s", from, to, body.String())
			w("250 OK queued")
		case cmd == "RSET":
			w("250 OK")
		case cmd == "QUIT":
			w("221 Bye")
			return
		default:
			w("250 OK")
		}
	}
}
