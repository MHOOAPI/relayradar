package api

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"relayradar/internal/config"
	"relayradar/internal/store"
)

// TestNotifyFailure checks the failure email body and manage link using a
// minimal in-process SMTP server.
func TestNotifyFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	got := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		write := func(s string) { fmt.Fprintf(conn, "%s\r\n", s) }
		write("220 test")
		var data strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if inData {
				if line == "." {
					inData = false
					got <- data.String()
					write("250 ok")
					continue
				}
				data.WriteString(line + "\n")
				continue
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				write("250-test")
				write("250 OK")
			case strings.HasPrefix(line, "HELO"):
				write("250 OK")
			case strings.HasPrefix(line, "MAIL FROM"), strings.HasPrefix(line, "RCPT TO"):
				write("250 OK")
			case line == "DATA":
				inData = true
				write("354 go ahead")
			case line == "QUIT":
				write("221 bye")
				return
			default:
				write("250 OK")
			}
		}
	}()

	srv := &Server{
		cfg: config.Config{
			SMTPAddr:      ln.Addr().String(),
			SMTPFrom:      "alerts@relayradar.local",
			PublicBaseURL: "http://127.0.0.1:8080",
		},
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	h := &alertHub{srv: srv}
	sub := &store.Subscription{ID: "sub-test", Channel: "email", Email: "user@example.com", Secret: "tok-123", URL: "https://example.com/hook"}

	h.notifyFailure(context.Background(), sub, errors.New("receiver returned HTTP 500"))

	select {
	case body := <-got:
		if !strings.Contains(body, "/alerts?id=sub-test&token=tok-123") {
			t.Fatalf("manage link missing from body:\n%s", body)
		}
		if !strings.Contains(body, "HTTP 500") {
			t.Fatalf("failure cause missing from body:\n%s", body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for SMTP message")
	}
}
