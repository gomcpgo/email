package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Save original env vars
	origEmail := os.Getenv("EMAIL_ADDRESS")
	origPassword := os.Getenv("EMAIL_APP_PASSWORD")
	defer func() {
		os.Setenv("EMAIL_ADDRESS", origEmail)
		os.Setenv("EMAIL_APP_PASSWORD", origPassword)
	}()

	// Test missing email address
	os.Unsetenv("EMAIL_ADDRESS")
	os.Unsetenv("EMAIL_APP_PASSWORD")
	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for missing EMAIL_ADDRESS")
	}

	// Test missing password
	os.Setenv("EMAIL_ADDRESS", "test@example.com")
	os.Unsetenv("EMAIL_APP_PASSWORD")
	_, err = LoadConfig()
	if err == nil {
		t.Error("Expected error for missing EMAIL_APP_PASSWORD")
	}

	// Test successful load with Gmail
	os.Setenv("EMAIL_ADDRESS", "test@gmail.com")
	os.Setenv("EMAIL_APP_PASSWORD", "test-password")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check Gmail auto-configuration
	if cfg.IMAPServer != "imap.gmail.com" {
		t.Errorf("Expected imap.gmail.com, got %s", cfg.IMAPServer)
	}
	if cfg.IMAPPort != 993 {
		t.Errorf("Expected port 993, got %d", cfg.IMAPPort)
	}
	if cfg.SMTPServer != "smtp.gmail.com" {
		t.Errorf("Expected smtp.gmail.com, got %s", cfg.SMTPServer)
	}
	if cfg.SMTPPort != 587 {
		t.Errorf("Expected port 587, got %d", cfg.SMTPPort)
	}
}

func TestConfigValidate(t *testing.T) {
	// Test valid config
	cfg := &Config{
		EmailAddress:  "test@example.com",
		EmailPassword: "password",
		IMAPServer:    "imap.example.com",
		IMAPPort:      993,
		SMTPServer:    "smtp.example.com",
		SMTPPort:      587,
	}
	
	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config failed validation: %v", err)
	}

	// Test missing email
	cfg.EmailAddress = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for missing email address")
	}
	cfg.EmailAddress = "test@example.com"

	// Test missing password
	cfg.EmailPassword = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for missing password")
	}
	cfg.EmailPassword = "password"

	// Test missing IMAP server
	cfg.IMAPServer = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for missing IMAP server")
	}
}