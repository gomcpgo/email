package config

import (
	"os"
	"testing"
)

// TestLoadConfig_NoAccounts verifies that LoadConfig succeeds with no accounts configured
func TestLoadConfig_NoAccounts(t *testing.T) {
	// Clear all environment variables
	clearEmailEnv(t)

	// Load config with no accounts
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig should succeed with no accounts, got error: %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig returned nil config")
	}

	if len(cfg.Accounts) != 0 {
		t.Errorf("Expected 0 accounts, got %d", len(cfg.Accounts))
	}

	if cfg.DefaultAccountID != "" {
		t.Errorf("Expected empty DefaultAccountID, got %s", cfg.DefaultAccountID)
	}
}

// TestLoadConfig_NoDefaultAccountID verifies that a default account is auto-selected
func TestLoadConfig_NoDefaultAccountID(t *testing.T) {
	// Clear all environment variables
	clearEmailEnv(t)

	// Set up one account but no DEFAULT_ACCOUNT_ID
	os.Setenv("ACCOUNT_WORK_EMAIL", "test@example.com")
	os.Setenv("ACCOUNT_WORK_PASSWORD", "testpass")
	os.Setenv("ACCOUNT_WORK_PROVIDER", "gmail")
	defer cleanupEmailEnv(t)

	// Load config
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Should auto-select the only account as default
	if cfg.DefaultAccountID != "WORK" {
		t.Errorf("Expected DefaultAccountID to be auto-set to WORK, got %s", cfg.DefaultAccountID)
	}

	if len(cfg.Accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(cfg.Accounts))
	}
}

// TestLoadConfig_MultipleAccountsNoDefault verifies default account selection with multiple accounts
func TestLoadConfig_MultipleAccountsNoDefault(t *testing.T) {
	// Clear all environment variables
	clearEmailEnv(t)

	// Set up multiple accounts but no DEFAULT_ACCOUNT_ID
	os.Setenv("ACCOUNT_WORK_EMAIL", "work@example.com")
	os.Setenv("ACCOUNT_WORK_PASSWORD", "workpass")
	os.Setenv("ACCOUNT_PERSONAL_EMAIL", "personal@example.com")
	os.Setenv("ACCOUNT_PERSONAL_PASSWORD", "personalpass")
	defer cleanupEmailEnv(t)

	// Load config
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Should auto-select one of the accounts as default
	if cfg.DefaultAccountID == "" {
		t.Error("Expected DefaultAccountID to be auto-set, got empty string")
	}

	if len(cfg.Accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(cfg.Accounts))
	}

	// Verify the default account exists
	if _, ok := cfg.Accounts[cfg.DefaultAccountID]; !ok {
		t.Errorf("Default account %s not found in accounts", cfg.DefaultAccountID)
	}
}

// TestLoadConfig_InvalidDefaultAccount verifies error when specified default doesn't exist
func TestLoadConfig_InvalidDefaultAccount(t *testing.T) {
	// Clear all environment variables
	clearEmailEnv(t)

	// Set up account
	os.Setenv("ACCOUNT_WORK_EMAIL", "work@example.com")
	os.Setenv("ACCOUNT_WORK_PASSWORD", "workpass")
	os.Setenv("DEFAULT_ACCOUNT_ID", "NONEXISTENT")
	defer cleanupEmailEnv(t)

	// Load config - should fail
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig should fail when DEFAULT_ACCOUNT_ID references non-existent account")
	}
}

// clearEmailEnv clears all email-related environment variables
func clearEmailEnv(t *testing.T) {
	for _, env := range os.Environ() {
		if len(env) > 8 && env[:8] == "ACCOUNT_" {
			parts := splitEnv(env)
			os.Unsetenv(parts[0])
		}
	}
	os.Unsetenv("DEFAULT_ACCOUNT_ID")
	os.Unsetenv("FILES_ROOT")
	os.Unsetenv("EMAIL_CACHE_MAX_SIZE")
	os.Unsetenv("EMAIL_MAX_ATTACHMENT_SIZE")
}

// cleanupEmailEnv removes test environment variables
func cleanupEmailEnv(t *testing.T) {
	clearEmailEnv(t)
}

// splitEnv splits "KEY=VALUE" into [KEY, VALUE]
func splitEnv(env string) []string {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env, ""}
}
