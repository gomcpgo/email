package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteAndReadAccountMetadata(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	metadataPath := filepath.Join(tmpDir, "metadata.yaml")

	// Write metadata
	err := WriteAccountMetadata(metadataPath, "TestAccount", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
	}

	// Read metadata
	metadata, err := ReadAccountMetadata(metadataPath)
	if err != nil {
		t.Fatalf("Failed to read metadata: %v", err)
	}

	// Verify
	if metadata.AccountID != "TestAccount" {
		t.Errorf("Expected AccountID 'TestAccount', got '%s'", metadata.AccountID)
	}
	if metadata.EmailAddress != "test@example.com" {
		t.Errorf("Expected EmailAddress 'test@example.com', got '%s'", metadata.EmailAddress)
	}
	if metadata.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if metadata.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestReadAccountMetadata_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	metadataPath := filepath.Join(tmpDir, "nonexistent.yaml")

	_, err := ReadAccountMetadata(metadataPath)
	if err == nil {
		t.Error("Expected error when reading nonexistent metadata file")
	}
}

func TestScanExistingFolders(t *testing.T) {
	tmpDir := t.TempDir()

	// Create folder structures with metadata
	folders := map[string]string{
		"Personal": "personal@example.com",
		"Business": "business@example.com",
	}

	for folderName, email := range folders {
		folderPath := filepath.Join(tmpDir, folderName)
		os.MkdirAll(folderPath, 0755)
		metadataPath := filepath.Join(folderPath, "metadata.yaml")
		WriteAccountMetadata(metadataPath, folderName, email)
	}

	// Scan folders
	scanned, err := ScanExistingFolders(tmpDir)
	if err != nil {
		t.Fatalf("Failed to scan folders: %v", err)
	}

	// Verify
	if len(scanned) != 2 {
		t.Errorf("Expected 2 folders, got %d", len(scanned))
	}

	for folderName, email := range folders {
		metadata, ok := scanned[folderName]
		if !ok {
			t.Errorf("Folder '%s' not found in scan results", folderName)
			continue
		}
		if metadata.EmailAddress != email {
			t.Errorf("Folder '%s': expected email '%s', got '%s'", folderName, email, metadata.EmailAddress)
		}
	}
}

func TestScanExistingFolders_NonExistentRoot(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "nonexistent")

	scanned, err := ScanExistingFolders(nonExistentPath)
	if err != nil {
		t.Fatalf("Should not error on nonexistent root: %v", err)
	}
	if len(scanned) != 0 {
		t.Errorf("Expected 0 folders, got %d", len(scanned))
	}
}

func TestDetectMigrations_AccountRenamed(t *testing.T) {
	tmpDir := t.TempDir()

	// Create folder with old name "Business"
	businessFolder := filepath.Join(tmpDir, "Business")
	os.MkdirAll(businessFolder, 0755)
	WriteAccountMetadata(filepath.Join(businessFolder, "metadata.yaml"), "Business", "business@example.com")

	// Current accounts show it was renamed to "Operations"
	currentAccounts := map[string]string{
		"Operations": "business@example.com",
		"Personal":   "personal@example.com",
	}

	// Detect migrations
	migrations, err := DetectMigrations(tmpDir, currentAccounts)
	if err != nil {
		t.Fatalf("Failed to detect migrations: %v", err)
	}

	// Verify migration detected
	if len(migrations) != 1 {
		t.Fatalf("Expected 1 migration, got %d", len(migrations))
	}

	migration := migrations[0]
	if migration.OldFolderName != "Business" {
		t.Errorf("Expected OldFolderName 'Business', got '%s'", migration.OldFolderName)
	}
	if migration.NewAccountID != "Operations" {
		t.Errorf("Expected NewAccountID 'Operations', got '%s'", migration.NewAccountID)
	}
	if migration.EmailAddress != "business@example.com" {
		t.Errorf("Expected EmailAddress 'business@example.com', got '%s'", migration.EmailAddress)
	}
}

func TestDetectMigrations_NoChange(t *testing.T) {
	tmpDir := t.TempDir()

	// Create folder "Personal" with matching metadata
	personalFolder := filepath.Join(tmpDir, "Personal")
	os.MkdirAll(personalFolder, 0755)
	WriteAccountMetadata(filepath.Join(personalFolder, "metadata.yaml"), "Personal", "personal@example.com")

	// Current accounts match folder name
	currentAccounts := map[string]string{
		"Personal": "personal@example.com",
	}

	// Detect migrations
	migrations, err := DetectMigrations(tmpDir, currentAccounts)
	if err != nil {
		t.Fatalf("Failed to detect migrations: %v", err)
	}

	// Verify no migration needed
	if len(migrations) != 0 {
		t.Errorf("Expected 0 migrations, got %d", len(migrations))
	}
}

func TestDetectMigrations_OrphanedFolder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create folder for deleted account
	oldFolder := filepath.Join(tmpDir, "OldAccount")
	os.MkdirAll(oldFolder, 0755)
	WriteAccountMetadata(filepath.Join(oldFolder, "metadata.yaml"), "OldAccount", "old@example.com")

	// Current accounts don't include this email
	currentAccounts := map[string]string{
		"Personal": "personal@example.com",
	}

	// Detect migrations
	migrations, err := DetectMigrations(tmpDir, currentAccounts)
	if err != nil {
		t.Fatalf("Failed to detect migrations: %v", err)
	}

	// Verify no migration (folder is orphaned)
	if len(migrations) != 0 {
		t.Errorf("Expected 0 migrations for orphaned folder, got %d", len(migrations))
	}
}

func TestDetectMigrations_MultipleConflicts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two folders with same email (conflict scenario)
	// This shouldn't happen normally, but we handle it
	folder1 := filepath.Join(tmpDir, "Business")
	os.MkdirAll(folder1, 0755)
	meta1Path := filepath.Join(folder1, "metadata.yaml")
	WriteAccountMetadata(meta1Path, "Business", "business@example.com")

	// Make folder2 newer by sleeping briefly
	time.Sleep(10 * time.Millisecond)

	folder2 := filepath.Join(tmpDir, "Business_Old")
	os.MkdirAll(folder2, 0755)
	meta2Path := filepath.Join(folder2, "metadata.yaml")
	WriteAccountMetadata(meta2Path, "Business_Old", "business@example.com")

	// Current accounts
	currentAccounts := map[string]string{
		"Operations": "business@example.com",
	}

	// Detect migrations
	migrations, err := DetectMigrations(tmpDir, currentAccounts)
	if err != nil {
		t.Fatalf("Failed to detect migrations: %v", err)
	}

	// Should only migrate the newest folder
	if len(migrations) != 1 {
		t.Fatalf("Expected 1 migration (newest folder), got %d", len(migrations))
	}

	// The newest folder should be Business_Old since we created it second
	if migrations[0].OldFolderName != "Business_Old" {
		t.Errorf("Expected newest folder 'Business_Old' to be migrated, got '%s'", migrations[0].OldFolderName)
	}
}

func TestExecuteMigration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old folder with content
	oldFolder := filepath.Join(tmpDir, "Business")
	os.MkdirAll(oldFolder, 0755)
	draftsDir := filepath.Join(oldFolder, "drafts")
	os.MkdirAll(draftsDir, 0755)

	// Create a test file
	testFile := filepath.Join(draftsDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	// Write metadata
	metadataPath := filepath.Join(oldFolder, "metadata.yaml")
	WriteAccountMetadata(metadataPath, "Business", "business@example.com")

	// Create migration plan
	plan := MigrationPlan{
		OldFolderName: "Business",
		NewAccountID:  "Operations",
		EmailAddress:  "business@example.com",
	}

	// Execute migration
	err := ExecuteMigration(tmpDir, plan)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify old folder is gone
	if _, err := os.Stat(oldFolder); !os.IsNotExist(err) {
		t.Error("Old folder should not exist after migration")
	}

	// Verify new folder exists
	newFolder := filepath.Join(tmpDir, "Operations")
	if _, err := os.Stat(newFolder); os.IsNotExist(err) {
		t.Fatal("New folder should exist after migration")
	}

	// Verify content was moved
	newTestFile := filepath.Join(newFolder, "drafts", "test.txt")
	content, err := os.ReadFile(newTestFile)
	if err != nil {
		t.Fatalf("Failed to read test file in new location: %v", err)
	}
	if string(content) != "test content" {
		t.Error("File content should be preserved")
	}

	// Verify metadata was updated
	newMetadataPath := filepath.Join(newFolder, "metadata.yaml")
	metadata, err := ReadAccountMetadata(newMetadataPath)
	if err != nil {
		t.Fatalf("Failed to read new metadata: %v", err)
	}
	if metadata.AccountID != "Operations" {
		t.Errorf("Metadata AccountID should be updated to 'Operations', got '%s'", metadata.AccountID)
	}
}

func TestExecuteMigration_TargetExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create both old and new folders
	oldFolder := filepath.Join(tmpDir, "Business")
	os.MkdirAll(oldFolder, 0755)
	WriteAccountMetadata(filepath.Join(oldFolder, "metadata.yaml"), "Business", "business@example.com")

	newFolder := filepath.Join(tmpDir, "Operations")
	os.MkdirAll(newFolder, 0755)

	// Create migration plan
	plan := MigrationPlan{
		OldFolderName: "Business",
		NewAccountID:  "Operations",
		EmailAddress:  "business@example.com",
	}

	// Execute migration - should fail
	err := ExecuteMigration(tmpDir, plan)
	if err == nil {
		t.Fatal("Migration should fail when target folder exists")
	}

	// Verify old folder still exists (rollback)
	if _, err := os.Stat(oldFolder); os.IsNotExist(err) {
		t.Error("Old folder should still exist after failed migration")
	}
}

func TestExecuteMigration_SourceNotExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create migration plan for nonexistent folder
	plan := MigrationPlan{
		OldFolderName: "NonExistent",
		NewAccountID:  "Operations",
		EmailAddress:  "business@example.com",
	}

	// Execute migration - should fail
	err := ExecuteMigration(tmpDir, plan)
	if err == nil {
		t.Fatal("Migration should fail when source folder doesn't exist")
	}
}
