package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Clear all account env vars
	for _, env := range os.Environ() {
		if len(env) > 8 && env[:8] == "ACCOUNT_" {
			parts := []rune(env)
			for i, c := range parts {
				if c == '=' {
					os.Unsetenv(string(parts[:i]))
					break
				}
			}
		}
	}
	os.Unsetenv("DEFAULT_ACCOUNT_ID")

	// Test missing account configuration - should now succeed
	cfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig should succeed with no accounts, got error: %v", err)
	}
	if len(cfg.Accounts) != 0 {
		t.Errorf("Expected 0 accounts, got %d", len(cfg.Accounts))
	}

	// Test successful load with Gmail account
	os.Setenv("ACCOUNT_Personal_EMAIL", "test@gmail.com")
	os.Setenv("ACCOUNT_Personal_PASSWORD", "test-password")
	os.Setenv("DEFAULT_ACCOUNT_ID", "Personal")

	cfg, err = LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check multi-account structure
	if len(cfg.Accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(cfg.Accounts))
	}

	acct, ok := cfg.Accounts["Personal"]
	if !ok {
		t.Fatal("Personal account not found")
	}

	// Check Gmail auto-configuration for the account
	if acct.IMAPServer != "imap.gmail.com" {
		t.Errorf("Expected imap.gmail.com, got %s", acct.IMAPServer)
	}
	if acct.IMAPPort != 993 {
		t.Errorf("Expected port 993, got %d", acct.IMAPPort)
	}
	if acct.SMTPServer != "smtp.gmail.com" {
		t.Errorf("Expected smtp.gmail.com, got %s", acct.SMTPServer)
	}
	if acct.SMTPPort != 587 {
		t.Errorf("Expected port 587, got %d", acct.SMTPPort)
	}

	// Cleanup
	os.Unsetenv("ACCOUNT_Personal_EMAIL")
	os.Unsetenv("ACCOUNT_Personal_PASSWORD")
	os.Unsetenv("DEFAULT_ACCOUNT_ID")
}

func TestMultiAccountConfig_Validate(t *testing.T) {
	// Test valid multi-account config
	cfg := &MultiAccountConfig{
		FilesRoot:         "/tmp/test",
		CacheMaxSize:      10485760,
		MaxAttachmentSize: 26214400,
		Accounts: map[string]*AccountConfig{
			"Personal": {
				AccountID:     "Personal",
				EmailAddress:  "test@example.com",
				EmailPassword: "password",
				IMAPServer:    "imap.example.com",
				IMAPPort:      993,
				SMTPServer:    "smtp.example.com",
				SMTPPort:      587,
			},
		},
		DefaultAccountID: "Personal",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config failed validation: %v", err)
	}

	// Test missing default account
	cfg.DefaultAccountID = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for missing default account")
	}
	cfg.DefaultAccountID = "Personal"

	// Test no accounts
	cfg.Accounts = map[string]*AccountConfig{}
	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for no accounts")
	}
}

func TestAccountConfig_ValidateForOperation(t *testing.T) {
	acct := &AccountConfig{
		AccountID:     "Test",
		EmailAddress:  "test@example.com",
		EmailPassword: "password",
		IMAPServer:    "imap.example.com",
		IMAPPort:      993,
		SMTPServer:    "smtp.example.com",
		SMTPPort:      587,
	}

	if err := acct.ValidateForOperation(); err != nil {
		t.Errorf("Valid account failed validation: %v", err)
	}

	// Test missing email
	acct.EmailAddress = ""
	if err := acct.ValidateForOperation(); err == nil {
		t.Error("Expected error for missing email address")
	}
	acct.EmailAddress = "test@example.com"

	// Test missing password
	acct.EmailPassword = ""
	if err := acct.ValidateForOperation(); err == nil {
		t.Error("Expected error for missing password")
	}
	acct.EmailPassword = "password"

	// Test missing IMAP server
	acct.IMAPServer = ""
	if err := acct.ValidateForOperation(); err == nil {
		t.Error("Expected error for missing IMAP server")
	}
}