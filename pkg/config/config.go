package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
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

	// Storage settings
	FilesRoot           string
	CacheMaxSize        int64
	MaxAttachmentSize   int64
	TimeoutSeconds      int
	Timeout             time.Duration

	// Derived paths
	DraftsDir      string
	CacheDir       string
	EmailCacheDir  string
	AttachmentDir  string
	MetadataFile   string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Provider:          "gmail",
		CacheMaxSize:      10485760,  // 10MB default
		MaxAttachmentSize: 26214400,  // 25MB default
		TimeoutSeconds:    120,        // 2 minutes default
		FilesRoot:         "/tmp/email-mcp",
	}

	// Required email account settings
	cfg.EmailAddress = os.Getenv("EMAIL_ADDRESS")
	if cfg.EmailAddress == "" {
		return nil, fmt.Errorf("EMAIL_ADDRESS environment variable is required")
	}

	cfg.EmailPassword = os.Getenv("EMAIL_APP_PASSWORD")
	if cfg.EmailPassword == "" {
		return nil, fmt.Errorf("EMAIL_APP_PASSWORD environment variable is required")
	}

	// Provider
	if provider := os.Getenv("EMAIL_PROVIDER"); provider != "" {
		cfg.Provider = provider
	}

	// Auto-configure for known providers
	switch cfg.Provider {
	case "gmail":
		cfg.IMAPServer = "imap.gmail.com"
		cfg.IMAPPort = 993
		cfg.SMTPServer = "smtp.gmail.com"
		cfg.SMTPPort = 587
	case "outlook":
		cfg.IMAPServer = "outlook.office365.com"
		cfg.IMAPPort = 993
		cfg.SMTPServer = "smtp-mail.outlook.com"
		cfg.SMTPPort = 587
	}

	// Override with explicit settings if provided
	if server := os.Getenv("EMAIL_IMAP_SERVER"); server != "" {
		cfg.IMAPServer = server
	}
	if port := os.Getenv("EMAIL_IMAP_PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_IMAP_PORT: %w", err)
		}
		cfg.IMAPPort = p
	}
	if server := os.Getenv("EMAIL_SMTP_SERVER"); server != "" {
		cfg.SMTPServer = server
	}
	if port := os.Getenv("EMAIL_SMTP_PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_SMTP_PORT: %w", err)
		}
		cfg.SMTPPort = p
	}

	// Storage settings
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
	if timeout := os.Getenv("EMAIL_TIMEOUT_SECONDS"); timeout != "" {
		t, err := strconv.Atoi(timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_TIMEOUT_SECONDS: %w", err)
		}
		cfg.TimeoutSeconds = t
	}

	// Set timeout duration
	cfg.Timeout = time.Duration(cfg.TimeoutSeconds) * time.Second

	// Validate required IMAP/SMTP settings
	if cfg.IMAPServer == "" {
		return nil, fmt.Errorf("EMAIL_IMAP_SERVER is required")
	}
	if cfg.IMAPPort == 0 {
		return nil, fmt.Errorf("EMAIL_IMAP_PORT is required")
	}
	if cfg.SMTPServer == "" {
		return nil, fmt.Errorf("EMAIL_SMTP_SERVER is required")
	}
	if cfg.SMTPPort == 0 {
		return nil, fmt.Errorf("EMAIL_SMTP_PORT is required")
	}

	// Setup derived paths
	cfg.DraftsDir = filepath.Join(cfg.FilesRoot, "drafts")
	cfg.CacheDir = filepath.Join(cfg.FilesRoot, "cache")
	cfg.EmailCacheDir = filepath.Join(cfg.CacheDir, "emails")
	cfg.AttachmentDir = filepath.Join(cfg.CacheDir, "attachments")
	cfg.MetadataFile = filepath.Join(cfg.FilesRoot, "metadata.yaml")

	// Create directories
	dirs := []string{cfg.DraftsDir, cfg.EmailCacheDir, cfg.AttachmentDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return cfg, nil
}

// Validate checks if configuration is valid
func (c *Config) Validate() error {
	if c.EmailAddress == "" {
		return fmt.Errorf("email address is required")
	}
	if c.EmailPassword == "" {
		return fmt.Errorf("email password is required")
	}
	if c.IMAPServer == "" || c.IMAPPort == 0 {
		return fmt.Errorf("IMAP server configuration is incomplete")
	}
	if c.SMTPServer == "" || c.SMTPPort == 0 {
		return fmt.Errorf("SMTP server configuration is incomplete")
	}
	return nil
}