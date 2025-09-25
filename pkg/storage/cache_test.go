package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheManager(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create cache manager with 1KB limit for testing
	cm := NewCacheManager(tempDir, 1024)

	// Test loading empty metadata
	metadata, err := cm.LoadMetadata()
	if err != nil {
		t.Fatalf("Failed to load empty metadata: %v", err)
	}
	if len(metadata.Entries) != 0 {
		t.Error("Expected empty entries for new metadata")
	}

	// Test adding entry
	err = cm.AddEntry("test1", "email", filepath.Join(tempDir, "test1.yaml"), 500)
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	// Test retrieving entry
	entry, err := cm.GetEntry("test1")
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}
	if entry.ID != "test1" {
		t.Errorf("Expected ID test1, got %s", entry.ID)
	}
	if entry.Type != "email" {
		t.Errorf("Expected type email, got %s", entry.Type)
	}
	if entry.Size != 500 {
		t.Errorf("Expected size 500, got %d", entry.Size)
	}

	// Test adding entry that exceeds limit
	err = cm.AddEntry("test2", "attachment", filepath.Join(tempDir, "test2.bin"), 600)
	if err != nil {
		t.Fatalf("Failed to add second entry: %v", err)
	}

	// Verify first entry was removed due to size limit
	metadata, err = cm.LoadMetadata()
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}
	
	// Should only have one entry due to size limit
	if len(metadata.Entries) != 1 {
		t.Errorf("Expected 1 entry after cleanup, got %d", len(metadata.Entries))
	}
	if metadata.Entries[0].ID != "test2" {
		t.Error("Expected test2 to remain after cleanup")
	}
}

func TestCacheCleanup(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "cache_cleanup_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cm := &CacheManager{
		rootDir:      tempDir,
		metadataFile: filepath.Join(tempDir, "metadata.yaml"),
		maxSize:      2048,
		maxAge:       24 * time.Hour,
	}

	// Create metadata with old and new entries
	oldTime := time.Now().Add(-25 * time.Hour)
	newTime := time.Now()

	metadata := &CacheMetadata{
		Version: 1,
		Entries: []CacheEntry{
			{
				ID:       "old",
				Type:     "email",
				Size:     500,
				CachedAt: oldTime,
				FilePath: filepath.Join(tempDir, "old.yaml"),
			},
			{
				ID:       "new",
				Type:     "email",
				Size:     500,
				CachedAt: newTime,
				FilePath: filepath.Join(tempDir, "new.yaml"),
			},
		},
		TotalSize: 1000,
	}

	// Run cleanup
	err = cm.cleanup(metadata)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify old entry was removed
	if len(metadata.Entries) != 1 {
		t.Errorf("Expected 1 entry after cleanup, got %d", len(metadata.Entries))
	}
	if metadata.Entries[0].ID != "new" {
		t.Error("Expected 'new' entry to remain")
	}
	if metadata.TotalSize != 500 {
		t.Errorf("Expected total size 500, got %d", metadata.TotalSize)
	}
}

func TestCacheInfo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_info_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cm := NewCacheManager(tempDir, 10240)

	// Add some entries
	cm.AddEntry("email1", "email", filepath.Join(tempDir, "e1.yaml"), 1000)
	cm.AddEntry("att1", "attachment", filepath.Join(tempDir, "a1.bin"), 2000)
	cm.AddEntry("email2", "email", filepath.Join(tempDir, "e2.yaml"), 1500)

	// Get cache info
	info, err := cm.GetCacheInfo()
	if err != nil {
		t.Fatalf("Failed to get cache info: %v", err)
	}

	if info.TotalSize != 4500 {
		t.Errorf("Expected total size 4500, got %d", info.TotalSize)
	}
	if info.MaxSize != 10240 {
		t.Errorf("Expected max size 10240, got %d", info.MaxSize)
	}
	if info.EntryCount != 3 {
		t.Errorf("Expected 3 entries, got %d", info.EntryCount)
	}
	if info.EmailCount != 2 {
		t.Errorf("Expected 2 emails, got %d", info.EmailCount)
	}
	if info.AttachmentCount != 1 {
		t.Errorf("Expected 1 attachment, got %d", info.AttachmentCount)
	}
}