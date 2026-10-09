package mailer

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestParseAddress(t *testing.T) {
	if got, err := ParseAddress("jane@example.com"); err != nil || got != "jane@example.com" {
		t.Errorf("ParseAddress(valid) = %q, %v", got, err)
	}
	for _, bad := range []string{"", "jane", "Jane <jane@example.com>", "a@b.com, c@d.com", "jane@example.com\r\nBcc: x@y.com", " jane@example.com"} {
		if _, err := ParseAddress(bad); err == nil {
			t.Errorf("ParseAddress(%q): expected error", bad)
		}
	}
}

func TestSMTP_Validate(t *testing.T) {
	ok := SMTP{Host: "smtp.example.com", Port: 587, Username: "u", Password: "p", From: "noreply@example.com"}
	if err := ok.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	cases := map[string]SMTP{
		"host":     {Port: 587, From: "noreply@example.com"},
		"port":     {Host: "smtp.example.com", Port: 0, From: "noreply@example.com"},
		"from":     {Host: "smtp.example.com", Port: 587, From: "nope"},
		"together": {Host: "smtp.example.com", Port: 587, From: "noreply@example.com", Username: "u"},
	}
	for want, cfg := range cases {
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Validate() = %v, want complaint about %s", err, want)
		}
	}
}

func TestBuildMessage(t *testing.T) {
	raw, err := buildMessage("noreply@qtidocs.dev", "dev@example.com", "Your secret", "line one\nline two\n", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	s := string(raw)
	for _, want := range []string{
		"From: noreply@qtidocs.dev\r\n",
		"To: dev@example.com\r\n",
		"Subject: Your secret\r\n",
		"Date: Fri, 02 Jan 2026 03:04:05 +0000\r\n",
		"Message-ID: <",
		"@qtidocs.dev>\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n",
		"\r\n\r\nline one\r\nline two\r\n",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("message missing %q:\n%s", want, s)
		}
	}
}

func TestBuildMessage_RejectsHeaderInjection(t *testing.T) {
	if _, err := buildMessage("a@b.com", "c@d.com", "hi\r\nBcc: x@y.com", "body", time.Now()); err == nil {
		t.Error("expected error for subject containing CRLF")
	}
}

func fakeSMTPServer(t *testing.T) (port int, received <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	out := make(chan string, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		reply := func(s string) { fmt.Fprintf(conn, "%s\r\n", s) }
		reply("220 fake ESMTP")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"):
				reply("250 fake")
			case strings.HasPrefix(cmd, "DATA"):
				reply("354 go ahead")
				var data strings.Builder
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if l == ".\r\n" {
						break
					}
					data.WriteString(l)
				}
				out <- data.String()
				reply("250 queued")
			case strings.HasPrefix(cmd, "QUIT"):
				reply("221 bye")
				return
			default:
				reply("250 ok")
			}
		}
	}()

	return ln.Addr().(*net.TCPAddr).Port, out
}

func TestSMTP_Send_Loopback(t *testing.T) {
	port, received := fakeSMTPServer(t)
	s := SMTP{Host: "127.0.0.1", Port: port, From: "noreply@qtidocs.dev", Timeout: 5 * time.Second}

	err := s.Send(context.Background(), Message{To: "dev@example.com", Subject: "Hello", Body: "secret-body\n"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	select {
	case data := <-received:
		if !strings.Contains(data, "To: dev@example.com\r\n") || !strings.Contains(data, "secret-body") {
			t.Errorf("unexpected message:\n%s", data)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server never received a message")
	}
}

func TestIsLoopback(t *testing.T) {
	for host, want := range map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true, "smtp.example.com": false, "10.0.0.1": false} {
		if got := isLoopback(host); got != want {
			t.Errorf("isLoopback(%q) = %v, want %v", host, got, want)
		}
	}
}
