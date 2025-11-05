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

// Storage handles file-based storage operations
type Storage struct {
	draftsDir     string
	emailCacheDir string
	cacheManager  *CacheManager
}

// NewStorage creates a new storage instance
func NewStorage(filesRoot string, cacheMaxSize int64) *Storage {
	s := &Storage{
		draftsDir:     filepath.Join(filesRoot, "drafts"),
		emailCacheDir: filepath.Join(filesRoot, "cache", "emails"),
		cacheManager:  NewCacheManager(filesRoot, cacheMaxSize),
	}
	
	// Create directories if they don't exist
	os.MkdirAll(s.draftsDir, 0755)
	os.MkdirAll(s.emailCacheDir, 0755)
	
	return s
}

// SaveEmail saves an email to cache
func (s *Storage) SaveEmail(e *email.Email) error {
	// Generate cache ID from Message-ID
	cacheID := s.generateEmailCacheID(e.MessageID)
	filename := fmt.Sprintf("msg_%s.yaml", cacheID)
	filePath := filepath.Join(s.emailCacheDir, filename)

	// Set cached time
	e.CachedAt = time.Now()

	// Marshal to YAML
	data, err := yaml.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal email: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write email cache: %w", err)
	}

	// Update cache metadata
	if err := s.cacheManager.AddEntry(cacheID, "email", filePath, int64(len(data))); err != nil {
		// Clean up file if metadata update fails
		os.Remove(filePath)
		return fmt.Errorf("failed to update cache metadata: %w", err)
	}

	return nil
}

// LoadEmail loads an email from cache
func (s *Storage) LoadEmail(messageID string) (*email.Email, error) {
	cacheID := s.generateEmailCacheID(messageID)
	
	// Check if cached
	entry, err := s.cacheManager.GetEntry(cacheID)
	if err != nil {
		return nil, fmt.Errorf("email not in cache: %w", err)
	}

	// Check if cache is stale (older than 1 day)
	if time.Since(entry.CachedAt) > 24*time.Hour {
		return nil, fmt.Errorf("cache entry expired")
	}

	// Read from file
	data, err := os.ReadFile(entry.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached email: %w", err)
	}

	// Unmarshal from YAML
	var e email.Email
	if err := yaml.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("failed to parse cached email: %w", err)
	}

	return &e, nil
}

// SaveDraft saves a draft email
func (s *Storage) SaveDraft(opts email.SendOptions) (string, error) {
	// Generate draft ID
	draftID := s.generateDraftID()
	filename := fmt.Sprintf("draft_%s.yaml", draftID)
	filePath := filepath.Join(s.draftsDir, filename)

	// Create draft structure
	draft := Draft{
		ID:               draftID,
		CreatedAt:        time.Now(),
		To:               opts.To,
		CC:               opts.CC,
		BCC:              opts.BCC,
		Subject:          opts.Subject,
		Body:             opts.Body,
		HTMLBody:         opts.HTMLBody,
		Attachments:      opts.Attachments,
		ReplyToMessageID: opts.ReplyToMessageID,
		References:       opts.References,
	}

	// Marshal to YAML
	data, err := yaml.Marshal(draft)
	if err != nil {
		return "", fmt.Errorf("failed to marshal draft: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write draft: %w", err)
	}

	return draftID, nil
}

// LoadDraft loads a draft by ID
func (s *Storage) LoadDraft(draftID string) (*Draft, error) {
	filename := fmt.Sprintf("draft_%s.yaml", draftID)
	filePath := filepath.Join(s.draftsDir, filename)

	// Read from file
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("draft not found: %s", draftID)
		}
		return nil, fmt.Errorf("failed to read draft: %w", err)
	}

	// Unmarshal from YAML
	var draft Draft
	if err := yaml.Unmarshal(data, &draft); err != nil {
		return nil, fmt.Errorf("failed to parse draft: %w", err)
	}

	return &draft, nil
}

// ListDrafts returns all draft IDs
func (s *Storage) ListDrafts() ([]DraftSummary, error) {
	files, err := os.ReadDir(s.draftsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read drafts directory: %w", err)
	}

	var drafts []DraftSummary
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "draft_") || !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}

		// Extract draft ID from filename
		draftID := strings.TrimSuffix(strings.TrimPrefix(file.Name(), "draft_"), ".yaml")
		
		// Load draft to get summary
		draft, err := s.LoadDraft(draftID)
		if err != nil {
			continue
		}

		drafts = append(drafts, DraftSummary{
			ID:        draft.ID,
			CreatedAt: draft.CreatedAt,
			Subject:   draft.Subject,
			To:        draft.To,
		})
	}

	return drafts, nil
}

// DeleteDraft deletes a draft by ID
func (s *Storage) DeleteDraft(draftID string) error {
	filename := fmt.Sprintf("draft_%s.yaml", draftID)
	filePath := filepath.Join(s.draftsDir, filename)
	
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("draft not found: %s", draftID)
		}
		return fmt.Errorf("failed to delete draft: %w", err)
	}
	
	return nil
}

// generateEmailCacheID generates a cache ID from a Message-ID
func (s *Storage) generateEmailCacheID(messageID string) string {
	// Clean up Message-ID (remove < > and special characters)
	clean := strings.Trim(messageID, "<>")
	clean = strings.ReplaceAll(clean, "@", "_at_")
	clean = strings.ReplaceAll(clean, ".", "_")
	
	// If too long, use hash
	if len(clean) > 50 {
		h := md5.New()
		h.Write([]byte(messageID))
		return fmt.Sprintf("%x", h.Sum(nil))
	}
	
	return clean
}

// generateDraftID generates a unique draft ID
func (s *Storage) generateDraftID() string {
	return fmt.Sprintf("%d_%x", time.Now().Unix(), time.Now().UnixNano()%1000000)
}

// Draft represents a saved email draft
type Draft struct {
	ID               string    `yaml:"id" json:"id"`
	CreatedAt        time.Time `yaml:"created_at" json:"created_at"`
	To               []string  `yaml:"to" json:"to"`
	CC               []string  `yaml:"cc,omitempty" json:"cc,omitempty"`
	BCC              []string  `yaml:"bcc,omitempty" json:"bcc,omitempty"`
	Subject          string    `yaml:"subject" json:"subject"`
	Body             string    `yaml:"body" json:"body"`
	HTMLBody         string    `yaml:"html_body,omitempty" json:"html_body,omitempty"`
	Attachments      []string  `yaml:"attachments,omitempty" json:"attachments,omitempty"`
	ReplyToMessageID string    `yaml:"reply_to_message_id,omitempty" json:"reply_to_message_id,omitempty"`
	References       []string  `yaml:"references,omitempty" json:"references,omitempty"`
}

// DraftSummary represents a draft summary for listing
type DraftSummary struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Subject   string    `json:"subject"`
	To        []string  `json:"to"`
}