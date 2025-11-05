package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// AccountMetadata stores persistent account information for migration tracking
type AccountMetadata struct {
	// Account identification
	AccountID    string    `yaml:"account_id"`
	EmailAddress string    `yaml:"email_address"`

	// Tracking
	CreatedAt  time.Time `yaml:"created_at"`
	UpdatedAt  time.Time `yaml:"updated_at"`
}

// WriteAccountMetadata writes account metadata to disk
func WriteAccountMetadata(metadataPath, accountID, emailAddress string) error {
	metadata := AccountMetadata{
		AccountID:    accountID,
		EmailAddress: emailAddress,
		UpdatedAt:    time.Now(),
	}

	// Set CreatedAt only if metadata doesn't exist yet
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		metadata.CreatedAt = time.Now()
	} else if err == nil {
		// Preserve existing CreatedAt if file exists
		existing, err := ReadAccountMetadata(metadataPath)
		if err == nil && !existing.CreatedAt.IsZero() {
			metadata.CreatedAt = existing.CreatedAt
		} else {
			metadata.CreatedAt = time.Now()
		}
	} else {
		return fmt.Errorf("failed to check metadata file: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to file
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// ReadAccountMetadata reads account metadata from disk
func ReadAccountMetadata(metadataPath string) (*AccountMetadata, error) {
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("metadata file not found: %w", err)
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata AccountMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// ScanExistingFolders scans filesRoot for existing account folders and their metadata
func ScanExistingFolders(filesRoot string) (map[string]*AccountMetadata, error) {
	folders := make(map[string]*AccountMetadata)

	// Check if filesRoot exists
	if _, err := os.Stat(filesRoot); os.IsNotExist(err) {
		// No folders exist yet, return empty map
		return folders, nil
	}

	// Read directory entries
	entries, err := os.ReadDir(filesRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read filesRoot directory: %w", err)
	}

	// For each subdirectory, try to read metadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		folderName := entry.Name()
		metadataPath := filepath.Join(filesRoot, folderName, "metadata.yaml")

		metadata, err := ReadAccountMetadata(metadataPath)
		if err != nil {
			// Folder exists but no valid metadata - this is an edge case
			// Log it but continue processing
			fmt.Fprintf(os.Stderr, "Warning: Folder %s exists but has no valid metadata (will be created on next account load)\n", folderName)
			continue
		}

		folders[folderName] = metadata
	}

	return folders, nil
}

// MigrationPlan represents a folder that needs to be migrated
type MigrationPlan struct {
	OldFolderName string
	NewAccountID  string
	EmailAddress  string
	Metadata      *AccountMetadata
}

// DetectMigrations scans for folders that need migration based on metadata vs current env vars
func DetectMigrations(filesRoot string, currentAccounts map[string]string) ([]MigrationPlan, error) {
	var migrations []MigrationPlan
	orphanedFolders := []string{}

	// Scan existing folders and read their metadata
	existingFolders, err := ScanExistingFolders(filesRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to scan existing folders: %w", err)
	}

	// Track which emails have already been matched to prevent conflicts
	emailToFolders := make(map[string][]string)
	for folderName, metadata := range existingFolders {
		emailToFolders[metadata.EmailAddress] = append(emailToFolders[metadata.EmailAddress], folderName)
	}

	// For each existing folder, check if migration is needed
	for folderName, metadata := range existingFolders {
		// Case 1: Folder name matches metadata accountID - no migration needed
		if folderName == metadata.AccountID {
			// Check if this account still exists in env vars with the same email
			if email, exists := currentAccounts[metadata.AccountID]; exists && email == metadata.EmailAddress {
				// All good, account unchanged
				continue
			}
		}

		// Case 2: Folder name doesn't match metadata accountID
		// This could be:
		// a) Account was renamed (email matches a different accountID)
		// b) Folder was manually renamed (shouldn't happen)
		// c) Account was deleted (email doesn't match any accountID)

		// Try to find matching account by email address
		var matchingAccountID string
		for accountID, email := range currentAccounts {
			if email == metadata.EmailAddress {
				matchingAccountID = accountID
				break
			}
		}

		// If we found a matching account ID and it's different from folder name, migrate
		if matchingAccountID != "" && matchingAccountID != folderName {
			// Check for conflicts: multiple folders with the same email
			if len(emailToFolders[metadata.EmailAddress]) > 1 {
				fmt.Fprintf(os.Stderr, "Warning: Multiple folders found for email %s: %v. Using newest folder for migration.\n",
					metadata.EmailAddress, emailToFolders[metadata.EmailAddress])

				// Use the folder with the most recent UpdatedAt timestamp
				newestFolder := folderName
				newestTime := metadata.UpdatedAt
				for _, otherFolder := range emailToFolders[metadata.EmailAddress] {
					if otherMeta, ok := existingFolders[otherFolder]; ok && otherMeta.UpdatedAt.After(newestTime) {
						newestFolder = otherFolder
						newestTime = otherMeta.UpdatedAt
					}
				}

				// Only migrate if this is the newest folder
				if folderName != newestFolder {
					orphanedFolders = append(orphanedFolders, folderName)
					continue
				}
			}

			migrations = append(migrations, MigrationPlan{
				OldFolderName: folderName,
				NewAccountID:  matchingAccountID,
				EmailAddress:  metadata.EmailAddress,
				Metadata:      metadata,
			})
		} else if matchingAccountID == "" {
			// No matching account found, folder is orphaned
			orphanedFolders = append(orphanedFolders, folderName)
		}
	}

	// Log orphaned folders
	if len(orphanedFolders) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: Found %d orphaned folder(s) with no matching account: %v\n",
			len(orphanedFolders), orphanedFolders)
		fmt.Fprintf(os.Stderr, "These folders will be preserved but not used. You can manually delete them if needed.\n")
	}

	return migrations, nil
}

// ExecuteMigration performs the actual folder rename and metadata update
func ExecuteMigration(filesRoot string, plan MigrationPlan) error {
	oldPath := filepath.Join(filesRoot, plan.OldFolderName)
	newPath := filepath.Join(filesRoot, plan.NewAccountID)

	// Safety check: verify old folder exists
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("source folder does not exist: %s", oldPath)
	}

	// Safety check: verify new folder doesn't already exist
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("target folder already exists: %s (cannot overwrite)", newPath)
	}

	// Rename the folder
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename folder from %s to %s: %w", oldPath, newPath, err)
	}

	// Update metadata with new account ID
	metadataPath := filepath.Join(newPath, "metadata.yaml")
	if err := WriteAccountMetadata(metadataPath, plan.NewAccountID, plan.EmailAddress); err != nil {
		// Try to rollback the rename if metadata update fails
		rollbackErr := os.Rename(newPath, oldPath)
		if rollbackErr != nil {
			return fmt.Errorf("failed to update metadata and rollback failed: metadata error: %w, rollback error: %v", err, rollbackErr)
		}
		return fmt.Errorf("failed to update metadata (rollback successful): %w", err)
	}

	return nil
}

// ExecuteAllMigrations performs all detected migrations
func ExecuteAllMigrations(filesRoot string, migrations []MigrationPlan) []error {
	var errors []error

	for _, plan := range migrations {
		if err := ExecuteMigration(filesRoot, plan); err != nil {
			errors = append(errors, fmt.Errorf("migration failed for %s -> %s: %w",
				plan.OldFolderName, plan.NewAccountID, err))
		}
	}

	return errors
}

