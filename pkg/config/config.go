package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AccountConfig represents configuration for a single email account
type AccountConfig struct {
	// Account identification
	AccountID string

	// Email account
	EmailAddress  string
	EmailPassword string
	Provider      string // gmail, outlook, or custom

	// IMAP settings
	IMAPServer string
	IMAPPort   int

	// SMTP settings
	SMTPServer string
	SMTPPort   int

	// Timeout settings
	TimeoutSeconds int
	Timeout        time.Duration

	// Derived paths (account-specific)
	DraftsDir     string
	CacheDir      string
	EmailCacheDir string
	AttachmentDir string
	MetadataFile  string
}

// MultiAccountConfig manages multiple email accounts
type MultiAccountConfig struct {
	// Global storage settings
	FilesRoot         string
	CacheMaxSize      int64
	MaxAttachmentSize int64

	// Account management
	Accounts         map[string]*AccountConfig
	DefaultAccountID string
}

// LoadConfig loads multi-account configuration from environment variables
func LoadConfig() (*MultiAccountConfig, error) {
	cfg := &MultiAccountConfig{
		FilesRoot:         "/tmp/email-mcp",
		CacheMaxSize:      10485760, // 10MB default
		MaxAttachmentSize: 26214400, // 25MB default
		Accounts:          make(map[string]*AccountConfig),
	}

	// Load global storage settings
	if root := os.Getenv("FILES_ROOT"); root != "" {
		cfg.FilesRoot = root
	}
	if size := os.Getenv("EMAIL_CACHE_MAX_SIZE"); size != "" {
		s, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_CACHE_MAX_SIZE: %w", err)
		}
		cfg.CacheMaxSize = s
	}
	if size := os.Getenv("EMAIL_MAX_ATTACHMENT_SIZE"); size != "" {
		s, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_MAX_ATTACHMENT_SIZE: %w", err)
		}
		cfg.MaxAttachmentSize = s
	}

	// Discover and load all accounts from environment variables
	accountIDs := discoverAccountIDs()
	if len(accountIDs) == 0 {
		return nil, fmt.Errorf("no email accounts configured: please set ACCOUNT_{name}_EMAIL environment variables")
	}

	// Build map of current accounts (accountID -> email) for migration detection
	currentAccounts := make(map[string]string)
	for _, accountID := range accountIDs {
		prefix := "ACCOUNT_" + accountID + "_"
		email := os.Getenv(prefix + "EMAIL")
		if email != "" {
			currentAccounts[accountID] = email
		}
	}

	// Detect and execute folder migrations before loading accounts
	migrations, err := DetectMigrations(cfg.FilesRoot, currentAccounts)
	if err != nil {
		return nil, fmt.Errorf("failed to detect migrations: %w", err)
	}

	if len(migrations) > 0 {
		fmt.Fprintf(os.Stderr, "Detected %d account folder migration(s)\n", len(migrations))
		migrationErrors := ExecuteAllMigrations(cfg.FilesRoot, migrations)
		if len(migrationErrors) > 0 {
			// Log migration errors but don't fail startup
			for _, err := range migrationErrors {
				fmt.Fprintf(os.Stderr, "Migration warning: %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "All migrations completed successfully\n")
		}
	}

	// Load all accounts
	for _, accountID := range accountIDs {
		acct, err := loadAccountConfig(accountID, cfg.FilesRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to load account %s: %w", accountID, err)
		}
		cfg.Accounts[accountID] = acct
	}

	// Load default account ID
	cfg.DefaultAccountID = os.Getenv("DEFAULT_ACCOUNT_ID")
	if cfg.DefaultAccountID == "" {
		return nil, fmt.Errorf("DEFAULT_ACCOUNT_ID environment variable is required")
	}

	// Validate default account exists
	if _, ok := cfg.Accounts[cfg.DefaultAccountID]; !ok {
		return nil, fmt.Errorf("default account %s not found in configured accounts", cfg.DefaultAccountID)
	}

	return cfg, nil
}

// discoverAccountIDs scans environment variables to find all configured accounts
func discoverAccountIDs() []string {
	accountSet := make(map[string]bool)
	prefix := "ACCOUNT_"
	suffix := "_EMAIL"

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		key := parts[0]

		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix) {
			// Extract account ID from ACCOUNT_{id}_EMAIL
			accountID := strings.TrimPrefix(key, prefix)
			accountID = strings.TrimSuffix(accountID, suffix)
			if accountID != "" {
				accountSet[accountID] = true
			}
		}
	}

	accountIDs := make([]string, 0, len(accountSet))
	for id := range accountSet {
		accountIDs = append(accountIDs, id)
	}
	return accountIDs
}

// loadAccountConfig loads configuration for a single account
func loadAccountConfig(accountID, filesRoot string) (*AccountConfig, error) {
	prefix := "ACCOUNT_" + accountID + "_"

	acct := &AccountConfig{
		AccountID:      accountID,
		Provider:       "gmail",       // default
		TimeoutSeconds: 120,           // 2 minutes default
	}

	// Load email credentials
	acct.EmailAddress = os.Getenv(prefix + "EMAIL")
	if acct.EmailAddress == "" {
		return nil, fmt.Errorf("missing %sEMAIL", prefix)
	}

	acct.EmailPassword = os.Getenv(prefix + "PASSWORD")
	if acct.EmailPassword == "" {
		return nil, fmt.Errorf("missing %sPASSWORD", prefix)
	}

	// Provider
	if provider := os.Getenv(prefix + "PROVIDER"); provider != "" {
		acct.Provider = provider
	}

	// Auto-configure for known providers
	switch acct.Provider {
	case "gmail":
		acct.IMAPServer = "imap.gmail.com"
		acct.IMAPPort = 993
		acct.SMTPServer = "smtp.gmail.com"
		acct.SMTPPort = 587
	case "outlook":
		acct.IMAPServer = "outlook.office365.com"
		acct.IMAPPort = 993
		acct.SMTPServer = "smtp-mail.outlook.com"
		acct.SMTPPort = 587
	default:
		// For custom providers, all settings must be explicitly provided
		acct.Provider = "custom"
	}

	// Override with explicit settings if provided
	if server := os.Getenv(prefix + "IMAP_SERVER"); server != "" {
		acct.IMAPServer = server
	}
	if port := os.Getenv(prefix + "IMAP_PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid %sIMAP_PORT: %w", prefix, err)
		}
		acct.IMAPPort = p
	}
	if server := os.Getenv(prefix + "SMTP_SERVER"); server != "" {
		acct.SMTPServer = server
	}
	if port := os.Getenv(prefix + "SMTP_PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid %sSMTP_PORT: %w", prefix, err)
		}
		acct.SMTPPort = p
	}
	if timeout := os.Getenv(prefix + "TIMEOUT_SECONDS"); timeout != "" {
		t, err := strconv.Atoi(timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid %sTIMEOUT_SECONDS: %w", prefix, err)
		}
		acct.TimeoutSeconds = t
	}

	// Set timeout duration
	acct.Timeout = time.Duration(acct.TimeoutSeconds) * time.Second

	// Validate required IMAP/SMTP settings
	if acct.IMAPServer == "" {
		return nil, fmt.Errorf("IMAP server not configured for account %s", accountID)
	}
	if acct.IMAPPort == 0 {
		return nil, fmt.Errorf("IMAP port not configured for account %s", accountID)
	}
	if acct.SMTPServer == "" {
		return nil, fmt.Errorf("SMTP server not configured for account %s", accountID)
	}
	if acct.SMTPPort == 0 {
		return nil, fmt.Errorf("SMTP port not configured for account %s", accountID)
	}

	// Setup account-specific paths
	accountRoot := filepath.Join(filesRoot, accountID)
	acct.DraftsDir = filepath.Join(accountRoot, "drafts")
	acct.CacheDir = filepath.Join(accountRoot, "cache")
	acct.EmailCacheDir = filepath.Join(acct.CacheDir, "emails")
	acct.AttachmentDir = filepath.Join(acct.CacheDir, "attachments")
	acct.MetadataFile = filepath.Join(accountRoot, "metadata.yaml")

	// Create directories
	dirs := []string{acct.DraftsDir, acct.EmailCacheDir, acct.AttachmentDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Write or update metadata for migration tracking
	if err := WriteAccountMetadata(acct.MetadataFile, acct.AccountID, acct.EmailAddress); err != nil {
		return nil, fmt.Errorf("failed to write account metadata: %w", err)
	}

	return acct, nil
}

// IsConfigured checks if email credentials are available
func (a *AccountConfig) IsConfigured() bool {
	return a.EmailAddress != "" && a.EmailPassword != ""
}

// ValidateForOperation checks if configuration is valid for email operations
func (a *AccountConfig) ValidateForOperation() error {
	if a.EmailAddress == "" {
		return fmt.Errorf("account %s: email address not configured", a.AccountID)
	}
	if a.EmailPassword == "" {
		return fmt.Errorf("account %s: email password not configured", a.AccountID)
	}
	if a.IMAPServer == "" || a.IMAPPort == 0 {
		return fmt.Errorf("account %s: IMAP server configuration is incomplete", a.AccountID)
	}
	if a.SMTPServer == "" || a.SMTPPort == 0 {
		return fmt.Errorf("account %s: SMTP server configuration is incomplete", a.AccountID)
	}
	return nil
}

// Validate checks if basic configuration is valid (used for startup)
func (m *MultiAccountConfig) Validate() error {
	if m.CacheMaxSize <= 0 {
		return fmt.Errorf("invalid cache size")
	}
	if len(m.Accounts) == 0 {
		return fmt.Errorf("no accounts configured")
	}
	if m.DefaultAccountID == "" {
		return fmt.Errorf("no default account specified")
	}
	return nil
}

// GetAccount returns the account config for the given ID, or default if empty
func (m *MultiAccountConfig) GetAccount(accountID string) (*AccountConfig, error) {
	if accountID == "" {
		accountID = m.DefaultAccountID
	}

	acct, ok := m.Accounts[accountID]
	if !ok {
		return nil, fmt.Errorf("account %s not found", accountID)
	}
	return acct, nil
}

// ListAccountIDs returns all configured account IDs
func (m *MultiAccountConfig) ListAccountIDs() []string {
	ids := make([]string, 0, len(m.Accounts))
	for id := range m.Accounts {
		ids = append(ids, id)
	}
	return ids
}