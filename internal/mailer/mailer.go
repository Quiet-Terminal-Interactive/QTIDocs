package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	To      string
	Subject string
	Body    string
}

type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	Timeout  time.Duration
}

func (s SMTP) Validate() error {
	var errs []string
	if s.Host == "" {
		errs = append(errs, "host is required")
	}
	if s.Port <= 0 || s.Port > 65535 {
		errs = append(errs, fmt.Sprintf("port %d is out of range", s.Port))
	}
	if _, err := ParseAddress(s.From); err != nil {
		errs = append(errs, "from: "+err.Error())
	}
	if (s.Username == "") != (s.Password == "") {
		errs = append(errs, "username and password must be set together")
	}
	if len(errs) > 0 {
		return fmt.Errorf("mailer: invalid SMTP config: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (s SMTP) Send(ctx context.Context, msg Message) error {
	to, err := ParseAddress(msg.To)
	if err != nil {
		return fmt.Errorf("mailer: recipient: %w", err)
	}
	from, err := ParseAddress(s.From)
	if err != nil {
		return fmt.Errorf("mailer: sender: %w", err)
	}

	raw, err := buildMessage(from, to, msg.Subject, msg.Body, time.Now())
	if err != nil {
		return err
	}

	timeout := s.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	tlsConfig := &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}

	var conn net.Conn
	dialer := &net.Dialer{}
	if s.Port == 465 {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("mailer: connecting to %s: %w", addr, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("mailer: starting SMTP session: %w", err)
	}
	defer func() { _ = c.Close() }()

	if s.Port != 465 {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("mailer: STARTTLS: %w", err)
			}
		} else if !isLoopback(s.Host) {
			return errors.New("mailer: server does not support STARTTLS; refusing to send credentials and secrets in plaintext")
		}
	}

	if s.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return fmt.Errorf("mailer: authenticating: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("mailer: MAIL FROM: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("mailer: RCPT TO: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("mailer: DATA: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		_ = w.Close()
		_ = c.Quit()
		return fmt.Errorf("mailer: writing message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: finishing message: %w", err)
	}
	return c.Quit()
}

func ParseAddress(s string) (string, error) {
	if strings.ContainsAny(s, "\r\n") {
		return "", fmt.Errorf("address %q contains a line break", s)
	}
	a, err := mail.ParseAddress(s)
	if err != nil {
		return "", fmt.Errorf("address %q is not a valid email address", s)
	}
	if a.Name != "" || a.Address != s {
		return "", fmt.Errorf("address %q must be a bare email address like jane@example.com", s)
	}
	return a.Address, nil
}

func buildMessage(from, to, subject, body string, now time.Time) ([]byte, error) {
	if strings.ContainsAny(subject, "\r\n") {
		return nil, errors.New("mailer: subject contains a line break")
	}
	id, err := messageID(from)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\n", now.Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: %s\r\n", id)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	body = strings.ReplaceAll(body, "\r\n", "\n")
	b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
	return b.Bytes(), nil
}

func messageID(from string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("mailer: generating message id: %w", err)
	}
	domain := "localhost"
	if i := strings.LastIndexByte(from, '@'); i >= 0 {
		domain = from[i+1:]
	}
	return "<" + hex.EncodeToString(buf) + "@" + domain + ">", nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
