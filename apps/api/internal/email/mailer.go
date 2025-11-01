package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Sender defines the contract for sending an email message.
type Sender interface {
	Send(ctx context.Context, msg *Message) error
	Enabled() bool
}

// Message represents an email to be sent.
type Message struct {
	To       []string
	Subject  string
	HTMLBody string
	TextBody string
	Template *TemplateData
	ReplyTo  string
	Headers  map[string]string
	Category Category
	Metadata map[string]string
}

// SMTPConfig contains the configuration needed to deliver email via SMTP.
type SMTPConfig struct {
	Host          string
	Port          int
	Username      string
	Password      string
	FromAddress   string
	FromName      string
	ReplyTo       string
	UseStartTLS   bool
	SkipTLSVerify bool
	Timeout       time.Duration
	LocalName     string
}

// SMTPMailer implements Sender using the standard library SMTP client.
type SMTPMailer struct {
	cfg SMTPConfig
}

// NoopSender is returned when SMTP is not configured. It logs and exits without sending.
type NoopSender struct {
	reason string
	once   sync.Once
}

// NewSenderFromEnv constructs an email Sender based on environment variables. When
// SMTP is not configured, a disabled sender is returned.
func NewSenderFromEnv() Sender {
	cfg, err := parseSMTPConfigFromEnv()
	if err != nil {
		log.Printf("[Email] SMTP disabled: %v", err)
		return &NoopSender{reason: err.Error()}
	}
	log.Printf("[Email] SMTP enabled: host=%s port=%d starttls=%t", cfg.Host, cfg.Port, cfg.UseStartTLS)
	return &SMTPMailer{cfg: cfg}
}

// Enabled returns true when SMTP is configured.
func (m *SMTPMailer) Enabled() bool { return true }

// Enabled returns false for the noop sender.
func (n *NoopSender) Enabled() bool { return false }

// Send logs the disabled state and returns nil.
func (n *NoopSender) Send(_ context.Context, msg *Message) error {
	n.once.Do(func() {
		log.Printf("[Email] Not sending email (disabled): %s", n.reason)
	})
	if msg != nil {
		log.Printf("[Email] Skipped message with subject %q destined for %s", msg.Subject, strings.Join(msg.To, ", "))
	}
	return nil
}

// Send delivers the email message via SMTP.
func (m *SMTPMailer) Send(ctx context.Context, msg *Message) error {
	if msg == nil {
		return errors.New("email message is nil")
	}

	if err := prepareMessage(msg); err != nil {
		return err
	}

	if msg.ReplyTo == "" {
		msg.ReplyTo = m.cfg.ReplyTo
	}

	raw, err := composeMessage(m.cfg, msg)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	dialer := &net.Dialer{Timeout: m.cfg.Timeout}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	localName := m.cfg.LocalName
	if localName == "" {
		localName = "api.obiente.local"
	}
	if err := client.Hello(localName); err != nil {
		log.Printf("[Email] EHLO failed: %v", err)
	}

	if m.cfg.UseStartTLS {
		tlsConfig := &tls.Config{ServerName: m.cfg.Host, InsecureSkipVerify: m.cfg.SkipTLSVerify}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if m.cfg.Username != "" {
		auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	if err := client.Mail(m.cfg.FromAddress); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}

	for _, rcpt := range msg.To {
		if rcpt == "" {
			continue
		}
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt to %s: %w", rcpt, err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp data close: %w", err)
	}

	if err := client.Quit(); err != nil {
		// If Quit fails, Close will handle the connection teardown.
		log.Printf("[Email] SMTP quit error: %v", err)
	}
	return nil
}

func prepareMessage(msg *Message) error {
	if msg.Template != nil {
		if msg.Template.Category == "" && msg.Category != "" {
			msg.Template.Category = msg.Category
		} else if msg.Template.Category != "" && msg.Category == "" {
			msg.Category = msg.Template.Category
		}
		if msg.Template.Subject != "" && msg.Subject == "" {
			msg.Subject = msg.Template.Subject
		}
		html, err := RenderHTML(*msg.Template)
		if err != nil {
			return err
		}
		msg.HTMLBody = html
		if msg.TextBody == "" {
			msg.TextBody = RenderText(*msg.Template)
		}
	}

	if len(msg.To) == 0 {
		return errors.New("email recipient list is empty")
	}
	if msg.Subject == "" {
		return errors.New("email subject is required")
	}
	if msg.HTMLBody == "" && msg.TextBody == "" {
		return errors.New("email body is empty")
	}

	if msg.Headers == nil {
		msg.Headers = make(map[string]string)
	}
	if msg.Metadata != nil {
		for key, value := range msg.Metadata {
			if key == "" || value == "" {
				continue
			}
			header := "X-Obiente-" + canonicalizeHeaderKey(key)
			msg.Headers[header] = value
		}
	}

	if msg.Category != "" {
		msg.Headers["X-Obiente-Category"] = string(msg.Category)
	}

	return nil
}

func composeMessage(cfg SMTPConfig, msg *Message) ([]byte, error) {
	boundary := fmt.Sprintf("mixed-%s", uuid.NewString())

	from := mail.Address{Name: cfg.FromName, Address: cfg.FromAddress}
	if from.Address == "" {
		return nil, errors.New("smtp from address is required")
	}

	toAddrs := make([]string, 0, len(msg.To))
	for _, addr := range msg.To {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		toAddrs = append(toAddrs, addr)
	}
	if len(toAddrs) == 0 {
		return nil, errors.New("smtp requires at least one recipient")
	}

	var buf bytes.Buffer

	encodedSubject := mime.QEncoding.Encode("utf-8", msg.Subject)
	dateHeader := time.Now().UTC().Format(time.RFC1123Z)
	messageID := fmt.Sprintf("<%s@%s>", uuid.NewString(), cfg.Host)

	buf.WriteString("From: ")
	buf.WriteString(from.String())
	buf.WriteString("\r\n")

	buf.WriteString("To: ")
	buf.WriteString(strings.Join(toAddrs, ", "))
	buf.WriteString("\r\n")

	if msg.ReplyTo != "" {
		buf.WriteString("Reply-To: ")
		buf.WriteString(msg.ReplyTo)
		buf.WriteString("\r\n")
	}

	buf.WriteString("Subject: ")
	buf.WriteString(encodedSubject)
	buf.WriteString("\r\n")

	buf.WriteString("Date: ")
	buf.WriteString(dateHeader)
	buf.WriteString("\r\n")

	buf.WriteString("Message-ID: ")
	buf.WriteString(messageID)
	buf.WriteString("\r\n")

	buf.WriteString("MIME-Version: 1.0\r\n")
	for key, value := range msg.Headers {
		if key == "" || value == "" {
			continue
		}
		buf.WriteString(key)
		buf.WriteString(": ")
		buf.WriteString(value)
		buf.WriteString("\r\n")
	}

	buf.WriteString("Content-Type: multipart/alternative; boundary=")
	buf.WriteString(boundary)
	buf.WriteString("\r\n\r\n")

	if msg.TextBody != "" {
		textBody := normalizeNewlines(msg.TextBody)
		buf.WriteString("--")
		buf.WriteString(boundary)
		buf.WriteString("\r\n")
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(textBody)
		if !strings.HasSuffix(textBody, "\r\n") {
			buf.WriteString("\r\n")
		}
		buf.WriteString("\r\n")
	}

	if msg.HTMLBody != "" {
		htmlBody := normalizeNewlines(msg.HTMLBody)
		buf.WriteString("--")
		buf.WriteString(boundary)
		buf.WriteString("\r\n")
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		buf.WriteString(htmlBody)
		if !strings.HasSuffix(htmlBody, "\r\n") {
			buf.WriteString("\r\n")
		}
		buf.WriteString("\r\n")
	}

	buf.WriteString("--")
	buf.WriteString(boundary)
	buf.WriteString("--\r\n")

	return buf.Bytes(), nil
}

func normalizeNewlines(input string) string {
	if input == "" {
		return ""
	}
	replaced := strings.ReplaceAll(input, "\r\n", "\n")
	replaced = strings.ReplaceAll(replaced, "\r", "\n")
	return strings.ReplaceAll(replaced, "\n", "\r\n")
}

func canonicalizeHeaderKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, " ", "-")
	key = strings.ReplaceAll(key, "_", "-")
	key = strings.ToLower(key)
	parts := strings.Split(key, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "-")
}

func parseSMTPConfigFromEnv() (SMTPConfig, error) {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return SMTPConfig{}, errors.New("SMTP_HOST is not set")
	}

	portStr := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	port := 587
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return SMTPConfig{}, fmt.Errorf("invalid SMTP_PORT: %w", err)
		}
		port = p
	}

	timeout := 10 * time.Second
	if t := strings.TrimSpace(os.Getenv("SMTP_TIMEOUT_SECONDS")); t != "" {
		seconds, err := strconv.Atoi(t)
		if err != nil {
			return SMTPConfig{}, fmt.Errorf("invalid SMTP_TIMEOUT_SECONDS: %w", err)
		}
		if seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	useStartTLS := true
	if v := strings.TrimSpace(os.Getenv("SMTP_USE_STARTTLS")); v != "" {
		useStartTLS = !(v == "false" || v == "0")
	}

	skipVerify := false
	if v := strings.TrimSpace(os.Getenv("SMTP_SKIP_TLS_VERIFY")); v != "" {
		skipVerify = v == "true" || v == "1"
	}

	cfg := SMTPConfig{
		Host:          host,
		Port:          port,
		Username:      strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		Password:      os.Getenv("SMTP_PASSWORD"),
		FromAddress:   strings.TrimSpace(os.Getenv("SMTP_FROM_ADDRESS")),
		FromName:      strings.TrimSpace(os.Getenv("SMTP_FROM_NAME")),
		ReplyTo:       strings.TrimSpace(os.Getenv("SMTP_REPLY_TO")),
		UseStartTLS:   useStartTLS,
		SkipTLSVerify: skipVerify,
		Timeout:       timeout,
		LocalName:     strings.TrimSpace(os.Getenv("SMTP_LOCAL_NAME")),
	}

	if cfg.FromAddress == "" {
		return SMTPConfig{}, errors.New("SMTP_FROM_ADDRESS is not set")
	}

	return cfg, nil
}
