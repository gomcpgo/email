package storage

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/prasanthmj/email/pkg/email"
	"gopkg.in/yaml.v3"
)

const (
	// CacheExpiry is how long cached emails remain valid
	CacheExpiry = 96 * time.Hour // 4 days
)

// CachedEmailMetadata stores email metadata separately from body content
type CachedEmailMetadata struct {
	MessageID   string             `yaml:"message_id" json:"message_id"`
	AccountID   string             `yaml:"account_id" json:"account_id"`
	Folder      string             `yaml:"folder" json:"folder"`
	From        string             `yaml:"from" json:"from"`
	To          []string           `yaml:"to" json:"to"`
	CC          []string           `yaml:"cc,omitempty" json:"cc,omitempty"`
	Subject     string             `yaml:"subject" json:"subject"`
	Date        time.Time          `yaml:"date" json:"date"`
	InReplyTo   string             `yaml:"in_reply_to,omitempty" json:"in_reply_to,omitempty"`
	References  []string           `yaml:"references,omitempty" json:"references,omitempty"`
	Attachments []email.Attachment `yaml:"attachments,omitempty" json:"attachments,omitempty"`
	CachedAt    time.Time          `yaml:"cached_at" json:"cached_at"`

	// Body size info
	TextBodySize      int64 `yaml:"text_body_size" json:"text_body_size"`
	HTMLBodySize      int64 `yaml:"html_body_size" json:"html_body_size"`
	ConvertedTextSize int64 `yaml:"converted_text_size,omitempty" json:"converted_text_size,omitempty"`
}

// EmailCacheInfo is returned by fetch_email to give LLM info about the cached email
type EmailCacheInfo struct {
	MessageID   string             `json:"message_id"`
	From        string             `json:"from"`
	To          []string           `json:"to"`
	CC          []string           `json:"cc,omitempty"`
	Subject     string             `json:"subject"`
	Date        time.Time          `json:"date"`
	InReplyTo   string             `json:"in_reply_to,omitempty"`
	References  []string           `json:"references,omitempty"`
	Attachments []email.Attachment `json:"attachments,omitempty"`
	Body        BodyInfo           `json:"body"`
}

// BodyInfo contains information about email body content
type BodyInfo struct {
	TextSize int64  `json:"text_size"`
	HTMLSize int64  `json:"html_size"`
	HasText  bool   `json:"has_text"`
	HasHTML  bool   `json:"has_html"`
	Preview  string `json:"preview"`
}

// ReadBodyResult is returned when reading email body content
type ReadBodyResult struct {
	Content    string `json:"content"`
	Format     string `json:"format"`      // "text" or "raw_html"
	Source     string `json:"source"`      // "text_body", "html_converted", "html_body", "none"
	TotalSize  int64  `json:"total_size"`
	Offset     int64  `json:"offset"`
	Limit      int64  `json:"limit"`
	Remaining  int64  `json:"remaining"`
	IsComplete bool   `json:"is_complete"`
}

// EmailCache handles caching of emails with separate body files
type EmailCache struct {
	cacheDir     string
	cacheManager *CacheManager
}

// NewEmailCache creates a new email cache instance
func NewEmailCache(filesRoot string, cacheMaxSize int64) *EmailCache {
	cacheDir := filepath.Join(filesRoot, "cache", "emails")
	os.MkdirAll(cacheDir, 0755)

	return &EmailCache{
		cacheDir:     cacheDir,
		cacheManager: NewCacheManager(filesRoot, cacheMaxSize),
	}
}

// generateCacheID creates a filesystem-safe cache ID from message ID
func (ec *EmailCache) generateCacheID(messageID string) string {
	// Clean up Message-ID (remove < > and special characters)
	clean := strings.Trim(messageID, "<>")
	clean = strings.ReplaceAll(clean, "@", "_at_")
	clean = strings.ReplaceAll(clean, ".", "_")
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, "\\", "_")

	// If too long, use hash
	if len(clean) > 50 {
		h := md5.New()
		h.Write([]byte(messageID))
		return fmt.Sprintf("%x", h.Sum(nil))
	}

	return clean
}

// getEmailDir returns the directory for a cached email
func (ec *EmailCache) getEmailDir(messageID string) string {
	cacheID := ec.generateCacheID(messageID)
	return filepath.Join(ec.cacheDir, cacheID)
}

// SaveEmail saves an email to cache with separate body files
func (ec *EmailCache) SaveEmail(e *email.Email, accountID string) (*CachedEmailMetadata, error) {
	emailDir := ec.getEmailDir(e.MessageID)

	// Create directory for this email
	if err := os.MkdirAll(emailDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create email cache dir: %w", err)
	}

	// Create metadata
	metadata := &CachedEmailMetadata{
		MessageID:    e.MessageID,
		AccountID:    accountID,
		Folder:       e.Folder,
		From:         e.From,
		To:           e.To,
		CC:           e.CC,
		Subject:      e.Subject,
		Date:         e.Date,
		InReplyTo:    e.InReplyTo,
		References:   e.References,
		Attachments:  e.Attachments,
		CachedAt:     time.Now(),
		TextBodySize: int64(len(e.Body)),
		HTMLBodySize: int64(len(e.HTMLBody)),
	}

	// Save text body if present
	if e.Body != "" {
		textPath := filepath.Join(emailDir, "body_text.txt")
		if err := os.WriteFile(textPath, []byte(e.Body), 0644); err != nil {
			return nil, fmt.Errorf("failed to write text body: %w", err)
		}
	}

	// Save HTML body if present
	if e.HTMLBody != "" {
		htmlPath := filepath.Join(emailDir, "body_html.txt")
		if err := os.WriteFile(htmlPath, []byte(e.HTMLBody), 0644); err != nil {
			return nil, fmt.Errorf("failed to write HTML body: %w", err)
		}

		// Pre-convert HTML to text and cache it
		if e.Body == "" {
			convertedText, err := email.ConvertHTMLToText(e.HTMLBody)
			if err == nil && convertedText != "" {
				convertedPath := filepath.Join(emailDir, "body_converted.txt")
				if err := os.WriteFile(convertedPath, []byte(convertedText), 0644); err == nil {
					metadata.ConvertedTextSize = int64(len(convertedText))
				}
			}
		}
	}

	// Save metadata
	metadataPath := filepath.Join(emailDir, "metadata.yaml")
	metadataBytes, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metadataPath, metadataBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	// Update cache manager
	totalSize := metadata.TextBodySize + metadata.HTMLBodySize + metadata.ConvertedTextSize + int64(len(metadataBytes))
	cacheID := ec.generateCacheID(e.MessageID)
	if err := ec.cacheManager.AddEntry(cacheID, "email", emailDir, totalSize); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to update cache metadata: %v\n", err)
	}

	return metadata, nil
}

// LoadMetadata loads email metadata from cache
func (ec *EmailCache) LoadMetadata(messageID string) (*CachedEmailMetadata, error) {
	emailDir := ec.getEmailDir(messageID)
	metadataPath := filepath.Join(emailDir, "metadata.yaml")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("email not in cache")
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata CachedEmailMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Check if cache is expired
	if time.Since(metadata.CachedAt) > CacheExpiry {
		return nil, fmt.Errorf("cache entry expired")
	}

	return &metadata, nil
}

// GetCacheInfo returns information about a cached email for the LLM
func (ec *EmailCache) GetCacheInfo(messageID string, previewLength int) (*EmailCacheInfo, error) {
	metadata, err := ec.LoadMetadata(messageID)
	if err != nil {
		return nil, err
	}

	// Generate preview
	preview := ec.generatePreview(messageID, metadata, previewLength)

	return &EmailCacheInfo{
		MessageID:   metadata.MessageID,
		From:        metadata.From,
		To:          metadata.To,
		CC:          metadata.CC,
		Subject:     metadata.Subject,
		Date:        metadata.Date,
		InReplyTo:   metadata.InReplyTo,
		References:  metadata.References,
		Attachments: metadata.Attachments,
		Body: BodyInfo{
			TextSize: metadata.TextBodySize,
			HTMLSize: metadata.HTMLBodySize,
			HasText:  metadata.TextBodySize > 0,
			HasHTML:  metadata.HTMLBodySize > 0,
			Preview:  preview,
		},
	}, nil
}

// generatePreview creates a text preview from the email body
func (ec *EmailCache) generatePreview(messageID string, metadata *CachedEmailMetadata, maxLength int) string {
	emailDir := ec.getEmailDir(messageID)

	// Try text body first
	if metadata.TextBodySize > 0 {
		textPath := filepath.Join(emailDir, "body_text.txt")
		content, err := ec.readFileChunk(textPath, 0, int64(maxLength))
		if err == nil {
			return content
		}
	}

	// Try converted HTML text
	if metadata.ConvertedTextSize > 0 {
		convertedPath := filepath.Join(emailDir, "body_converted.txt")
		content, err := ec.readFileChunk(convertedPath, 0, int64(maxLength))
		if err == nil {
			return content
		}
	}

	// As last resort, try to convert HTML on the fly
	if metadata.HTMLBodySize > 0 {
		htmlPath := filepath.Join(emailDir, "body_html.txt")
		htmlContent, err := os.ReadFile(htmlPath)
		if err == nil {
			converted, err := email.ConvertHTMLToText(string(htmlContent))
			if err == nil {
				// Cache the converted text for future use
				convertedPath := filepath.Join(emailDir, "body_converted.txt")
				os.WriteFile(convertedPath, []byte(converted), 0644)

				if len(converted) > maxLength {
					return converted[:maxLength]
				}
				return converted
			}
		}
	}

	return ""
}

// ReadBody reads email body content with pagination support
func (ec *EmailCache) ReadBody(messageID string, format string, offset, limit int64) (*ReadBodyResult, error) {
	metadata, err := ec.LoadMetadata(messageID)
	if err != nil {
		return nil, err
	}

	emailDir := ec.getEmailDir(messageID)

	// Handle format selection
	if format == "raw_html" {
		return ec.readRawHTML(emailDir, metadata, offset, limit)
	}

	// Default: text format
	return ec.readText(emailDir, metadata, offset, limit)
}

// readText reads text content (from text body or converted HTML)
func (ec *EmailCache) readText(emailDir string, metadata *CachedEmailMetadata, offset, limit int64) (*ReadBodyResult, error) {
	// Try text body first
	if metadata.TextBodySize > 0 {
		textPath := filepath.Join(emailDir, "body_text.txt")
		return ec.readBodyFile(textPath, "text", "text_body", metadata.TextBodySize, offset, limit)
	}

	// Try converted HTML
	if metadata.ConvertedTextSize > 0 {
		convertedPath := filepath.Join(emailDir, "body_converted.txt")
		return ec.readBodyFile(convertedPath, "text", "html_converted", metadata.ConvertedTextSize, offset, limit)
	}

	// Convert HTML on the fly if needed
	if metadata.HTMLBodySize > 0 {
		htmlPath := filepath.Join(emailDir, "body_html.txt")
		htmlContent, err := os.ReadFile(htmlPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read HTML body: %w", err)
		}

		converted, err := email.ConvertHTMLToText(string(htmlContent))
		if err != nil {
			return nil, fmt.Errorf("failed to convert HTML: %w", err)
		}

		// Cache the converted text
		convertedPath := filepath.Join(emailDir, "body_converted.txt")
		os.WriteFile(convertedPath, []byte(converted), 0644)

		// Update metadata with converted size
		metadata.ConvertedTextSize = int64(len(converted))
		metadataPath := filepath.Join(emailDir, "metadata.yaml")
		metadataBytes, _ := yaml.Marshal(metadata)
		os.WriteFile(metadataPath, metadataBytes, 0644)

		// Now read from the converted file
		return ec.readBodyFile(convertedPath, "text", "html_converted", metadata.ConvertedTextSize, offset, limit)
	}

	// No body content
	return &ReadBodyResult{
		Content:    "",
		Format:     "text",
		Source:     "none",
		TotalSize:  0,
		Offset:     0,
		Limit:      limit,
		Remaining:  0,
		IsComplete: true,
	}, nil
}

// readRawHTML reads raw HTML content
func (ec *EmailCache) readRawHTML(emailDir string, metadata *CachedEmailMetadata, offset, limit int64) (*ReadBodyResult, error) {
	if metadata.HTMLBodySize == 0 {
		return &ReadBodyResult{
			Content:    "",
			Format:     "raw_html",
			Source:     "none",
			TotalSize:  0,
			Offset:     0,
			Limit:      limit,
			Remaining:  0,
			IsComplete: true,
		}, nil
	}

	htmlPath := filepath.Join(emailDir, "body_html.txt")
	return ec.readBodyFile(htmlPath, "raw_html", "html_body", metadata.HTMLBodySize, offset, limit)
}

// readBodyFile reads a chunk from a body file
func (ec *EmailCache) readBodyFile(filePath, format, source string, totalSize, offset, limit int64) (*ReadBodyResult, error) {
	content, err := ec.readFileChunk(filePath, offset, limit)
	if err != nil {
		return nil, err
	}

	remaining := totalSize - offset - int64(len(content))
	if remaining < 0 {
		remaining = 0
	}

	return &ReadBodyResult{
		Content:    content,
		Format:     format,
		Source:     source,
		TotalSize:  totalSize,
		Offset:     offset,
		Limit:      limit,
		Remaining:  remaining,
		IsComplete: remaining == 0,
	}, nil
}

// readFileChunk reads a chunk of a file starting at offset with max length limit
func (ec *EmailCache) readFileChunk(filePath string, offset, limit int64) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Seek to offset
	if offset > 0 {
		_, err = file.Seek(offset, 0)
		if err != nil {
			return "", fmt.Errorf("failed to seek: %w", err)
		}
	}

	// Read up to limit bytes
	buffer := make([]byte, limit)
	n, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return "", fmt.Errorf("failed to read: %w", err)
	}

	return string(buffer[:n]), nil
}

// IsCached checks if an email is in cache and not expired
func (ec *EmailCache) IsCached(messageID string) bool {
	_, err := ec.LoadMetadata(messageID)
	return err == nil
}
