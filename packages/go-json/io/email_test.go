package io

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmail_Send_Validation_MissingTo(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"}, nil)
	_, err := m.send(map[string]any{"subject": "Hi", "body": "Hello"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'to' is required")
}

func TestEmail_Send_Validation_MissingSubject(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"}, nil)
	_, err := m.send(map[string]any{"to": "user@test.com", "body": "Hello"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'subject' is required")
}

func TestEmail_Send_Validation_MissingBody(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"}, nil)
	_, err := m.send(map[string]any{"to": "user@test.com", "subject": "Hi"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'body' or 'html' is required")
}

func TestEmail_Send_Validation_MissingHost(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{From: "test@test.com"}, nil)
	_, err := m.send(map[string]any{"to": "user@test.com", "subject": "Hi", "body": "Hello"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP host not configured")
}

func TestEmail_Send_Validation_MissingFrom(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{Host: "localhost", Port: 25}, nil)
	_, err := m.send(map[string]any{"to": "user@test.com", "subject": "Hi", "body": "Hello"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'from' is required")
}

func TestEmail_Send_Validation_MissingArgs(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"}, nil)
	_, err := m.send()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "options map required")
}

func TestEmail_Send_Validation_NonMapArg(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"}, nil)
	_, err := m.send("not a map")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "argument must be a map")
}

func TestEmail_Security_BlockedDomain(t *testing.T) {
	m := NewEmailModuleWithConfig(
		EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"},
		&EmailSecurityConfig{BlockedDomains: []string{"evil.com"}},
	)
	_, err := m.send(map[string]any{"to": "hacker@evil.com", "subject": "Hi", "body": "Hello"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestEmail_Security_BlockedDomain_CaseInsensitive(t *testing.T) {
	m := NewEmailModuleWithConfig(
		EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"},
		&EmailSecurityConfig{BlockedDomains: []string{"Evil.COM"}},
	)
	_, err := m.send(map[string]any{"to": "hacker@evil.com", "subject": "Hi", "body": "Hello"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestEmail_Security_MaxRecipients(t *testing.T) {
	m := NewEmailModuleWithConfig(
		EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"},
		&EmailSecurityConfig{MaxRecipients: 2},
	)
	_, err := m.send(map[string]any{
		"to":      []any{"a@test.com", "b@test.com"},
		"cc":      "c@test.com",
		"subject": "Hi",
		"body":    "Hello",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many recipients")
}

func TestEmail_Security_AllowedRecipients(t *testing.T) {
	m := NewEmailModuleWithConfig(
		EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"},
		&EmailSecurityConfig{AllowedRecipients: []string{"*@company.com"}},
	)

	_, err := m.send(map[string]any{"to": "user@company.com", "subject": "Hi", "body": "Hello"})
	if err != nil {
		assert.NotContains(t, err.Error(), "not in allowed list")
	}

	_, err = m.send(map[string]any{"to": "user@external.com", "subject": "Hi", "body": "Hello"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in allowed list")
}

func TestEmail_Security_MaxBodySize(t *testing.T) {
	m := NewEmailModuleWithConfig(
		EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"},
		&EmailSecurityConfig{MaxBodySize: 10},
	)
	_, err := m.send(map[string]any{
		"to":      "user@test.com",
		"subject": "Hi",
		"body":    "This body is definitely longer than 10 bytes",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "body size")
}

func TestEmail_Security_InvalidRecipientAddress(t *testing.T) {
	m := NewEmailModuleWithConfig(
		EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"},
		nil,
	)
	_, err := m.send(map[string]any{"to": "invalid-no-at-sign", "subject": "Hi", "body": "Hello"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid recipient address")
}

func TestEmail_ExtractRecipients(t *testing.T) {
	assert.Equal(t, []string{"a@b.com"}, extractRecipients("a@b.com"))
	assert.Equal(t, []string{"a@b.com", "c@d.com"}, extractRecipients([]any{"a@b.com", "c@d.com"}))
	assert.Nil(t, extractRecipients(nil))
	assert.Nil(t, extractRecipients(""))
	assert.Equal(t, []string{"x@y.com"}, extractRecipients([]string{"x@y.com"}))
	assert.Nil(t, extractRecipients(123))
}

func TestEmail_BuildMessage_PlainText(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{}, nil)
	msg := m.buildMessage("from@test.com", []string{"to@test.com"}, nil, "Subject", "Body text", "", "")
	msgStr := string(msg)
	assert.Contains(t, msgStr, "From: from@test.com")
	assert.Contains(t, msgStr, "To: to@test.com")
	assert.Contains(t, msgStr, "Subject: Subject")
	assert.Contains(t, msgStr, "text/plain")
	assert.Contains(t, msgStr, "Body text")
	assert.NotContains(t, msgStr, "MIME-Version")
}

func TestEmail_BuildMessage_HTML(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{}, nil)
	msg := m.buildMessage("from@test.com", []string{"to@test.com"}, nil, "Subject", "", "<h1>Hello</h1>", "")
	msgStr := string(msg)
	assert.Contains(t, msgStr, "text/html")
	assert.Contains(t, msgStr, "MIME-Version: 1.0")
	assert.Contains(t, msgStr, "<h1>Hello</h1>")
}

func TestEmail_BuildMessage_WithCC(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{}, nil)
	msg := m.buildMessage("from@test.com", []string{"to@test.com"}, []string{"cc@test.com"}, "Subject", "Body", "", "reply@test.com")
	msgStr := string(msg)
	assert.Contains(t, msgStr, "Cc: cc@test.com")
	assert.Contains(t, msgStr, "Reply-To: reply@test.com")
}

func TestEmail_BuildMessage_MultipleRecipients(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{}, nil)
	msg := m.buildMessage("from@test.com", []string{"a@test.com", "b@test.com"}, nil, "Subject", "Body", "", "")
	msgStr := string(msg)
	assert.Contains(t, msgStr, "To: a@test.com, b@test.com")
}

func TestEmail_Name(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{}, nil)
	assert.Equal(t, "email", m.Name())
}

func TestEmail_Functions(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{}, nil)
	fns := m.Functions()
	assert.Contains(t, fns, "send")
	assert.Len(t, fns, 1)
}

func TestEmail_SetConfig(t *testing.T) {
	m := NewEmailModuleWithConfig(EmailConfig{}, nil)
	m.SetConfig(map[string]any{
		"host":     "smtp.example.com",
		"port":     float64(465),
		"username": "user",
		"password": "pass",
		"from":     "noreply@example.com",
		"tls":      false,
	})
	assert.Equal(t, "smtp.example.com", m.config.Host)
	assert.Equal(t, 465, m.config.Port)
	assert.Equal(t, "user", m.config.Username)
	assert.Equal(t, "pass", m.config.Password)
	assert.Equal(t, "noreply@example.com", m.config.From)
	assert.Equal(t, false, m.config.TLS)
}

func TestEmail_MatchEmailGlob(t *testing.T) {
	assert.True(t, matchEmailGlob("*", "anything"))
	assert.True(t, matchEmailGlob("*@company.com", "user@company.com"))
	assert.False(t, matchEmailGlob("*@company.com", "user@other.com"))
	assert.True(t, matchEmailGlob("admin@*", "admin@anything.com"))
	assert.False(t, matchEmailGlob("admin@*", "user@anything.com"))
	assert.True(t, matchEmailGlob("exact@match.com", "exact@match.com"))
	assert.False(t, matchEmailGlob("exact@match.com", "other@match.com"))
}

func TestEmail_Send_WithFromOverride(t *testing.T) {
	m := NewEmailModuleWithConfig(
		EmailConfig{Host: "localhost", Port: 25, From: "default@test.com"},
		nil,
	)
	_, err := m.send(map[string]any{
		"to":      "user@test.com",
		"from":    "override@test.com",
		"subject": "Hi",
		"body":    "Hello",
	})
	if err != nil {
		assert.NotContains(t, err.Error(), "'from' is required")
	}
}

func TestEmail_Send_EmptyToArray(t *testing.T) {
	m := NewEmailModuleWithConfig(
		EmailConfig{Host: "localhost", Port: 25, From: "test@test.com"},
		nil,
	)
	_, err := m.send(map[string]any{
		"to":      []any{},
		"subject": "Hi",
		"body":    "Hello",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'to' is required")
}
