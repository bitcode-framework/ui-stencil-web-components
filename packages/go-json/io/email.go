package io

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

// EmailConfig holds SMTP connection settings.
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      bool
}

// EmailSecurityConfig controls email sending limits.
type EmailSecurityConfig struct {
	AllowedRecipients []string `json:"allowed_recipients"` // Glob patterns for allowed recipients. Empty = all allowed.
	BlockedDomains    []string `json:"blocked_domains"`    // Blocked recipient domains.
	MaxBodySize       int64    `json:"max_body_size"`      // Max body size in bytes. Default: 1048576 (1MB). 0 = unlimited.
	MaxRecipients     int      `json:"max_recipients"`     // Max total recipients (to + cc + bcc). Default: 50. 0 = unlimited.
}

// EmailModule provides SMTP email sending.
type EmailModule struct {
	config   EmailConfig
	security *EmailSecurityConfig
}

// NewEmailModule creates a new email I/O module with explicit config.
func NewEmailModule(security *SecurityConfig) *EmailModule {
	config := EmailConfig{}

	config.Host = os.Getenv("SMTP_HOST")
	if p, err := strconv.Atoi(os.Getenv("SMTP_PORT")); err == nil {
		config.Port = p
	} else {
		config.Port = 587
	}
	config.Username = os.Getenv("SMTP_USER")
	config.Password = os.Getenv("SMTP_PASSWORD")
	config.From = os.Getenv("SMTP_FROM")
	if os.Getenv("SMTP_TLS") == "false" {
		config.TLS = false
	} else {
		config.TLS = true
	}

	sec := &EmailSecurityConfig{
		MaxBodySize:   1048576,
		MaxRecipients: 50,
	}
	if security != nil {
		cfg := &security.Email
		if cfg.MaxBodySize != 0 {
			sec.MaxBodySize = cfg.MaxBodySize
		}
		if cfg.MaxRecipients != 0 {
			sec.MaxRecipients = cfg.MaxRecipients
		}
		if len(cfg.AllowedRecipients) > 0 {
			sec.AllowedRecipients = cfg.AllowedRecipients
		}
		if len(cfg.BlockedDomains) > 0 {
			sec.BlockedDomains = cfg.BlockedDomains
		}
	}

	return &EmailModule{config: config, security: sec}
}

// NewEmailModuleWithConfig creates an email module with explicit SMTP config (for testing/embedding).
func NewEmailModuleWithConfig(config EmailConfig, security *EmailSecurityConfig) *EmailModule {
	if config.Port == 0 {
		config.Port = 587
	}
	if security == nil {
		security = &EmailSecurityConfig{
			MaxBodySize:   1048576,
			MaxRecipients: 50,
		}
	}
	return &EmailModule{config: config, security: security}
}

func (m *EmailModule) Name() string { return "email" }

func (m *EmailModule) Functions() map[string]any {
	return map[string]any{
		"send": m.send,
	}
}

func (m *EmailModule) SetConfig(cfg map[string]any) {
	if host, ok := cfg["host"].(string); ok && host != "" {
		m.config.Host = host
	}
	if port, ok := toFloat64Val(cfg["port"]); ok && port > 0 {
		m.config.Port = int(port)
	}
	if user, ok := cfg["username"].(string); ok {
		m.config.Username = user
	}
	if pass, ok := cfg["password"].(string); ok {
		m.config.Password = pass
	}
	if from, ok := cfg["from"].(string); ok {
		m.config.From = from
	}
	if tlsVal, ok := cfg["tls"].(bool); ok {
		m.config.TLS = tlsVal
	}
}

func (m *EmailModule) send(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("email.send: options map required")
	}
	opts, ok := params[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("email.send: argument must be a map")
	}

	to := extractRecipients(opts["to"])
	cc := extractRecipients(opts["cc"])
	bcc := extractRecipients(opts["bcc"])
	subject, _ := opts["subject"].(string)
	body, _ := opts["body"].(string)
	html, _ := opts["html"].(string)
	from, _ := opts["from"].(string)
	replyTo, _ := opts["replyTo"].(string)

	if from == "" {
		from = m.config.From
	}

	if len(to) == 0 {
		return nil, fmt.Errorf("email.send: 'to' is required")
	}
	if subject == "" {
		return nil, fmt.Errorf("email.send: 'subject' is required")
	}
	if body == "" && html == "" {
		return nil, fmt.Errorf("email.send: 'body' or 'html' is required")
	}
	if from == "" {
		return nil, fmt.Errorf("email.send: 'from' is required (set via config or SMTP_FROM env)")
	}
	if m.config.Host == "" {
		return nil, fmt.Errorf("email.send: SMTP host not configured (set via config or SMTP_HOST env)")
	}

	allRecipients := make([]string, 0, len(to)+len(cc)+len(bcc))
	allRecipients = append(allRecipients, to...)
	allRecipients = append(allRecipients, cc...)
	allRecipients = append(allRecipients, bcc...)
	if m.security.MaxRecipients > 0 && len(allRecipients) > m.security.MaxRecipients {
		return nil, fmt.Errorf("email.send: too many recipients (%d, max %d)", len(allRecipients), m.security.MaxRecipients)
	}
	for _, addr := range allRecipients {
		if err := m.checkRecipientAllowed(addr); err != nil {
			return nil, err
		}
	}
	if m.security.MaxBodySize > 0 {
		bodyLen := int64(len(body) + len(html))
		if bodyLen > m.security.MaxBodySize {
			return nil, fmt.Errorf("email.send: body size (%d) exceeds limit (%d)", bodyLen, m.security.MaxBodySize)
		}
	}

	msg := m.buildMessage(from, to, cc, subject, body, html, replyTo)
	return nil, m.sendSMTP(from, allRecipients, msg)
}

func (m *EmailModule) buildMessage(from string, to, cc []string, subject, body, html, replyTo string) []byte {
	var msg strings.Builder

	msg.WriteString("From: " + from + "\r\n")
	msg.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	if len(cc) > 0 {
		msg.WriteString("Cc: " + strings.Join(cc, ", ") + "\r\n")
	}
	if replyTo != "" {
		msg.WriteString("Reply-To: " + replyTo + "\r\n")
	}
	msg.WriteString("Subject: " + subject + "\r\n")

	if html != "" {
		msg.WriteString("MIME-Version: 1.0\r\n")
		msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(html)
	} else {
		msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(body)
	}

	return []byte(msg.String())
}

func (m *EmailModule) sendSMTP(from string, recipients []string, msg []byte) error {
	addr := net.JoinHostPort(m.config.Host, strconv.Itoa(m.config.Port))

	var auth smtp.Auth
	if m.config.Username != "" {
		auth = smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	}

	if m.config.TLS {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("email: connection failed: %w", err)
		}

		client, err := smtp.NewClient(conn, m.config.Host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("email: SMTP client failed: %w", err)
		}
		defer client.Close()

		tlsConfig := &tls.Config{ServerName: m.config.Host}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("email: STARTTLS failed: %w", err)
		}

		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("email: auth failed: %w", err)
			}
		}

		if err := client.Mail(from); err != nil {
			return fmt.Errorf("email: MAIL FROM failed: %w", err)
		}
		for _, rcpt := range recipients {
			if err := client.Rcpt(rcpt); err != nil {
				return fmt.Errorf("email: RCPT TO <%s> failed: %w", rcpt, err)
			}
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("email: DATA failed: %w", err)
		}
		_, err = w.Write(msg)
		if err != nil {
			return fmt.Errorf("email: write failed: %w", err)
		}
		return w.Close()
	}

	return smtp.SendMail(addr, auth, from, recipients, msg)
}

func (m *EmailModule) checkRecipientAllowed(addr string) error {
	parts := strings.Split(addr, "@")
	if len(parts) != 2 {
		return fmt.Errorf("email.send: invalid recipient address '%s'", addr)
	}
	domain := strings.ToLower(parts[1])

	for _, blocked := range m.security.BlockedDomains {
		if strings.ToLower(blocked) == domain {
			return fmt.Errorf("email.send: recipient domain '%s' is blocked", domain)
		}
	}

	if len(m.security.AllowedRecipients) > 0 {
		allowed := false
		for _, pattern := range m.security.AllowedRecipients {
			if matchEmailGlob(strings.ToLower(pattern), strings.ToLower(addr)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("email.send: recipient '%s' not in allowed list", addr)
		}
	}

	return nil
}

func extractRecipients(v any) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		if val == "" {
			return nil
		}
		return []string{val}
	case []any:
		var result []string
		for _, item := range val {
			if s, ok := item.(string); ok && s != "" {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return val
	}
	return nil
}

// matchEmailGlob performs simple glob matching (supports * prefix/suffix).
func matchEmailGlob(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(value, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(value, pattern[:len(pattern)-1])
	}
	return pattern == value
}
