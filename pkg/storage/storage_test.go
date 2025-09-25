package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prasanthmj/email/pkg/email"
)

func TestSaveAndLoadEmail(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create storage
	s := NewStorage(tempDir, 10485760)

	// Create test email
	testEmail := &email.Email{
		MessageID: "<test123@example.com>",
		Folder:    "INBOX",
		From:      "sender@example.com",
		To:        []string{"recipient@example.com"},
		Subject:   "Test Subject",
		Date:      time.Now(),
		Body:      "Test body content",
		Attachments: []email.Attachment{
			{
				Filename: "test.pdf",
				Size:     1024,
			},
		},
	}

	// Save email
	err = s.SaveEmail(testEmail)
	if err != nil {
		t.Fatalf("Failed to save email: %v", err)
	}

	// Load email
	loaded, err := s.LoadEmail(testEmail.MessageID)
	if err != nil {
		t.Fatalf("Failed to load email: %v", err)
	}

	// Verify loaded email
	if loaded.MessageID != testEmail.MessageID {
		t.Errorf("Expected MessageID %s, got %s", testEmail.MessageID, loaded.MessageID)
	}
	if loaded.Subject != testEmail.Subject {
		t.Errorf("Expected subject %s, got %s", testEmail.Subject, loaded.Subject)
	}
	if loaded.Body != testEmail.Body {
		t.Errorf("Expected body %s, got %s", testEmail.Body, loaded.Body)
	}
	if len(loaded.Attachments) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(loaded.Attachments))
	}
}

func TestDraftOperations(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "draft_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	s := NewStorage(tempDir, 10485760)

	// Create test draft
	opts := email.SendOptions{
		To:      []string{"recipient@example.com"},
		CC:      []string{"cc@example.com"},
		Subject: "Test Draft",
		Body:    "This is a test draft",
	}

	// Save draft
	draftID, err := s.SaveDraft(opts)
	if err != nil {
		t.Fatalf("Failed to save draft: %v", err)
	}
	if draftID == "" {
		t.Error("Expected non-empty draft ID")
	}

	// Load draft
	draft, err := s.LoadDraft(draftID)
	if err != nil {
		t.Fatalf("Failed to load draft: %v", err)
	}
	if draft.Subject != opts.Subject {
		t.Errorf("Expected subject %s, got %s", opts.Subject, draft.Subject)
	}
	if draft.Body != opts.Body {
		t.Errorf("Expected body %s, got %s", opts.Body, draft.Body)
	}

	// List drafts
	drafts, err := s.ListDrafts()
	if err != nil {
		t.Fatalf("Failed to list drafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Errorf("Expected 1 draft, got %d", len(drafts))
	}
	if drafts[0].ID != draftID {
		t.Errorf("Expected draft ID %s, got %s", draftID, drafts[0].ID)
	}

	// Delete draft
	err = s.DeleteDraft(draftID)
	if err != nil {
		t.Fatalf("Failed to delete draft: %v", err)
	}

	// Verify deletion
	_, err = s.LoadDraft(draftID)
	if err == nil {
		t.Error("Expected error loading deleted draft")
	}
}

func TestGenerateEmailCacheID(t *testing.T) {
	s := &Storage{}

	tests := []struct {
		messageID string
		maxLen    int
	}{
		{"<simple@example.com>", 50},
		{"<CADsK8=very-long-message-id-that-exceeds-fifty-characters@mail.gmail.com>", 32}, // MD5 hash length
		{"<test@domain.com>", 50},
	}

	for _, test := range tests {
		cacheID := s.generateEmailCacheID(test.messageID)
		if len(cacheID) > test.maxLen {
			t.Errorf("Cache ID too long for %s: got %d chars, max %d", test.messageID, len(cacheID), test.maxLen)
		}
		// Ensure no problematic characters
		if filepath.Base(cacheID) != cacheID {
			t.Errorf("Cache ID contains path separators: %s", cacheID)
		}
	}
}

func TestGenerateDraftID(t *testing.T) {
	s := &Storage{}
	
	// Generate multiple IDs
	id1 := s.generateDraftID()
	time.Sleep(10 * time.Millisecond) // Ensure different timestamp
	id2 := s.generateDraftID()
	
	if id1 == "" {
		t.Error("Generated empty draft ID")
	}
	if id1 == id2 {
		t.Error("Generated duplicate draft IDs")
	}
}