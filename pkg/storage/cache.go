package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// CacheMetadata tracks cache entries
type CacheMetadata struct {
	Version     int          `yaml:"cache_version"`
	TotalSize   int64        `yaml:"total_size_bytes"`
	Entries     []CacheEntry `yaml:"entries"`
}

// CacheEntry represents a cached item
type CacheEntry struct {
	ID         string    `yaml:"id"`
	Type       string    `yaml:"type"` // "email" or "attachment"
	Size       int64     `yaml:"size_bytes"`
	CachedAt   time.Time `yaml:"cached_at"`
	AccessedAt time.Time `yaml:"accessed_at"`
	FilePath   string    `yaml:"file_path"`
}

// CacheManager manages the file cache
type CacheManager struct {
	rootDir      string
	metadataFile string
	maxSize      int64
	maxAge       time.Duration
}

// NewCacheManager creates a new cache manager
func NewCacheManager(rootDir string, maxSize int64) *CacheManager {
	cm := &CacheManager{
		rootDir:      rootDir,
		metadataFile: filepath.Join(rootDir, "cache", "cache_metadata.yaml"),
		maxSize:      maxSize,
		maxAge:       24 * time.Hour, // 1 day
	}

	// Migrate old cache metadata if it exists
	cm.migrateOldMetadata()

	return cm
}

// migrateOldMetadata migrates cache metadata from old location to new location
// Old location: {rootDir}/metadata.yaml
// New location: {rootDir}/cache/cache_metadata.yaml
func (cm *CacheManager) migrateOldMetadata() {
	oldPath := filepath.Join(cm.rootDir, "metadata.yaml")
	newPath := cm.metadataFile

	// Check if old metadata exists
	data, err := os.ReadFile(oldPath)
	if err != nil {
		// Old metadata doesn't exist, nothing to migrate
		return
	}

	// Try to parse as cache metadata
	var metadata CacheMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		// Not valid cache metadata, leave it alone (probably account metadata)
		return
	}

	// Check if it has cache_version field (distinguishes cache metadata from account metadata)
	if metadata.Version == 0 {
		// Doesn't look like cache metadata, leave it alone
		return
	}

	// Ensure cache directory exists
	cacheDir := filepath.Dir(newPath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		// Can't create directory, skip migration
		return
	}

	// Check if new location already exists
	if _, err := os.Stat(newPath); err == nil {
		// New metadata already exists, don't overwrite
		// Remove old file to avoid confusion
		os.Remove(oldPath)
		return
	}

	// Move the file to new location
	if err := os.Rename(oldPath, newPath); err != nil {
		// If rename fails, try copy+delete
		if writeErr := os.WriteFile(newPath, data, 0644); writeErr == nil {
			os.Remove(oldPath)
		}
	}
}

// LoadMetadata loads cache metadata from disk
func (cm *CacheManager) LoadMetadata() (*CacheMetadata, error) {
	data, err := os.ReadFile(cm.metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty metadata if file doesn't exist
			return &CacheMetadata{
				Version: 1,
				Entries: []CacheEntry{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata CacheMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// SaveMetadata saves cache metadata to disk
func (cm *CacheManager) SaveMetadata(metadata *CacheMetadata) error {
	data, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Ensure directory exists
	metadataDir := filepath.Dir(cm.metadataFile)
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	if err := os.WriteFile(cm.metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// AddEntry adds a new cache entry
func (cm *CacheManager) AddEntry(id, entryType, filePath string, size int64) error {
	metadata, err := cm.LoadMetadata()
	if err != nil {
		return err
	}

	// Check if entry already exists
	for i, entry := range metadata.Entries {
		if entry.ID == id {
			// Update existing entry
			metadata.Entries[i].AccessedAt = time.Now()
			return cm.SaveMetadata(metadata)
		}
	}

	// Add new entry
	entry := CacheEntry{
		ID:         id,
		Type:       entryType,
		Size:       size,
		CachedAt:   time.Now(),
		AccessedAt: time.Now(),
		FilePath:   filePath,
	}
	
	metadata.Entries = append(metadata.Entries, entry)
	metadata.TotalSize += size

	// Check if cleanup is needed
	if metadata.TotalSize > cm.maxSize {
		if err := cm.cleanup(metadata); err != nil {
			return err
		}
	}

	return cm.SaveMetadata(metadata)
}

// GetEntry retrieves a cache entry and updates access time
func (cm *CacheManager) GetEntry(id string) (*CacheEntry, error) {
	metadata, err := cm.LoadMetadata()
	if err != nil {
		return nil, err
	}

	for i, entry := range metadata.Entries {
		if entry.ID == id {
			// Update access time
			metadata.Entries[i].AccessedAt = time.Now()
			cm.SaveMetadata(metadata)
			return &entry, nil
		}
	}

	return nil, fmt.Errorf("cache entry not found: %s", id)
}

// cleanup removes old or excess cache entries
func (cm *CacheManager) cleanup(metadata *CacheMetadata) error {
	now := time.Now()
	var validEntries []CacheEntry
	var totalSize int64

	// First, remove entries older than 1 day
	for _, entry := range metadata.Entries {
		age := now.Sub(entry.CachedAt)
		if age < cm.maxAge {
			validEntries = append(validEntries, entry)
			totalSize += entry.Size
		} else {
			// Delete the file
			os.Remove(entry.FilePath)
		}
	}

	// If still over limit, remove oldest entries
	if totalSize > cm.maxSize {
		// Sort by cached time (oldest first)
		sort.Slice(validEntries, func(i, j int) bool {
			return validEntries[i].CachedAt.Before(validEntries[j].CachedAt)
		})

		// Remove entries until under limit
		for totalSize > cm.maxSize && len(validEntries) > 0 {
			entry := validEntries[0]
			validEntries = validEntries[1:]
			totalSize -= entry.Size
			os.Remove(entry.FilePath)
		}
	}

	metadata.Entries = validEntries
	metadata.TotalSize = totalSize
	return nil
}

// ClearCache removes all cache entries
func (cm *CacheManager) ClearCache() error {
	metadata, err := cm.LoadMetadata()
	if err != nil {
		return err
	}

	// Delete all cached files
	for _, entry := range metadata.Entries {
		os.Remove(entry.FilePath)
	}

	// Reset metadata
	metadata.Entries = []CacheEntry{}
	metadata.TotalSize = 0

	return cm.SaveMetadata(metadata)
}

// GetCacheInfo returns cache statistics
func (cm *CacheManager) GetCacheInfo() (CacheInfo, error) {
	metadata, err := cm.LoadMetadata()
	if err != nil {
		return CacheInfo{}, err
	}

	now := time.Now()
	var emailCount, attachmentCount int
	var oldestEntry, newestEntry time.Time

	for _, entry := range metadata.Entries {
		switch entry.Type {
		case "email":
			emailCount++
		case "attachment":
			attachmentCount++
		}

		if oldestEntry.IsZero() || entry.CachedAt.Before(oldestEntry) {
			oldestEntry = entry.CachedAt
		}
		if newestEntry.IsZero() || entry.CachedAt.After(newestEntry) {
			newestEntry = entry.CachedAt
		}
	}

	return CacheInfo{
		TotalSize:       metadata.TotalSize,
		MaxSize:         cm.maxSize,
		EntryCount:      len(metadata.Entries),
		EmailCount:      emailCount,
		AttachmentCount: attachmentCount,
		OldestEntry:     oldestEntry,
		NewestEntry:     newestEntry,
		CurrentTime:     now,
	}, nil
}

// CacheInfo represents cache statistics
type CacheInfo struct {
	TotalSize       int64     `json:"total_size_bytes"`
	MaxSize         int64     `json:"max_size_bytes"`
	EntryCount      int       `json:"entry_count"`
	EmailCount      int       `json:"email_count"`
	AttachmentCount int       `json:"attachment_count"`
	OldestEntry     time.Time `json:"oldest_entry"`
	NewestEntry     time.Time `json:"newest_entry"`
	CurrentTime     time.Time `json:"current_time"`
}