package email

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/prasanthmj/email/pkg/config"
)

// AttachmentFetcher handles attachment operations
type AttachmentFetcher struct {
	config     *config.Config
	imapClient *IMAPClient
}

// NewAttachmentFetcher creates a new attachment fetcher
func NewAttachmentFetcher(cfg *config.Config, imapClient *IMAPClient) *AttachmentFetcher {
	return &AttachmentFetcher{
		config:     cfg,
		imapClient: imapClient,
	}
}

// FetchAttachments fetches attachments from an email
func (af *AttachmentFetcher) FetchAttachments(messageID string, attachmentNames []string, fetchAll bool) ([]AttachmentResult, error) {
	c, err := af.imapClient.connect()
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	// Find the email in any folder
	attachments, err := af.searchAndFetchAttachments(c, messageID, attachmentNames, fetchAll)
	if err != nil {
		return nil, err
	}

	return attachments, nil
}

// AttachmentResult represents a fetched attachment
type AttachmentResult struct {
	Filename string `json:"filename"`
	CacheID  string `json:"cache_id"`
	Size     int64  `json:"size"`
	Saved    bool   `json:"saved"`
}

// searchAndFetchAttachments searches for an email and fetches its attachments
func (af *AttachmentFetcher) searchAndFetchAttachments(c *client.Client, messageID string, attachmentNames []string, fetchAll bool) ([]AttachmentResult, error) {
	// Try common folders first
	commonFolders := []string{"INBOX", "Sent", "[Gmail]/Sent Mail", "Sent Items", "[Gmail]/All Mail"}
	
	for _, folder := range commonFolders {
		results, err := af.fetchAttachmentsFromFolder(c, folder, messageID, attachmentNames, fetchAll)
		if err == nil && len(results) > 0 {
			return results, nil
		}
	}

	// If not found in common folders, search all folders
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	for m := range mailboxes {
		results, err := af.fetchAttachmentsFromFolder(c, m.Name, messageID, attachmentNames, fetchAll)
		if err == nil && len(results) > 0 {
			return results, nil
		}
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to search folders: %w", err)
	}

	return nil, fmt.Errorf("email not found: %s", messageID)
}

// fetchAttachmentsFromFolder fetches attachments from a specific folder
func (af *AttachmentFetcher) fetchAttachmentsFromFolder(c *client.Client, folder, messageID string, attachmentNames []string, fetchAll bool) ([]AttachmentResult, error) {
	mbox, err := c.Select(folder, true) // read-only
	if err != nil {
		return nil, err
	}

	if mbox.Messages == 0 {
		return nil, fmt.Errorf("folder empty")
	}

	// Search by Message-ID header
	criteria := imap.NewSearchCriteria()
	criteria.Header.Set("Message-ID", messageID)
	
	seqNums, err := c.Search(criteria)
	if err != nil {
		return nil, err
	}

	if len(seqNums) == 0 {
		return nil, fmt.Errorf("not found")
	}

	// Fetch the message
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(seqNums[0])

	messages := make(chan *imap.Message, 1)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}
	
	go func() {
		if err := c.Fetch(seqSet, items, messages); err != nil {
			// Log error
		}
	}()

	msg := <-messages
	if msg == nil {
		return nil, fmt.Errorf("failed to fetch message")
	}

	// Parse the message to extract attachments
	r := msg.GetBody(&imap.BodySectionName{})
	if r == nil {
		return nil, fmt.Errorf("failed to get message body")
	}

	mr, err := mail.CreateReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	var results []AttachmentResult
	
	// Create a map of requested attachment names for quick lookup
	requestedMap := make(map[string]bool)
	for _, name := range attachmentNames {
		requestedMap[strings.ToLower(name)] = true
	}

	// Extract attachments
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		switch h := p.Header.(type) {
		case *mail.AttachmentHeader:
			filename, err := h.Filename()
			if err != nil || filename == "" {
				continue
			}

			// Check if we should fetch this attachment
			shouldFetch := fetchAll || requestedMap[strings.ToLower(filename)]
			if !shouldFetch {
				continue
			}

			// Read attachment content
			content, err := io.ReadAll(p.Body)
			if err != nil {
				continue
			}

			// Check size limit
			if int64(len(content)) > af.config.MaxAttachmentSize {
				results = append(results, AttachmentResult{
					Filename: filename,
					Size:     int64(len(content)),
					Saved:    false,
					CacheID:  "",
				})
				continue
			}

			// Generate cache ID
			cacheID := af.generateCacheID(filename, content)
			
			// Save to cache
			cachePath := filepath.Join(af.config.AttachmentDir, cacheID)
			err = os.WriteFile(cachePath, content, 0644)
			if err != nil {
				results = append(results, AttachmentResult{
					Filename: filename,
					Size:     int64(len(content)),
					Saved:    false,
					CacheID:  "",
				})
				continue
			}

			results = append(results, AttachmentResult{
				Filename: filename,
				CacheID:  cacheID,
				Size:     int64(len(content)),
				Saved:    true,
			})
		}
	}

	if len(results) == 0 && !fetchAll && len(attachmentNames) > 0 {
		return nil, fmt.Errorf("requested attachments not found")
	}

	return results, nil
}

// generateCacheID generates a unique cache ID for an attachment
func (af *AttachmentFetcher) generateCacheID(filename string, content []byte) string {
	// Use MD5 hash of content plus filename for uniqueness
	h := md5.New()
	h.Write([]byte(filename))
	h.Write(content)
	hash := fmt.Sprintf("%x", h.Sum(nil))
	
	// Get file extension
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".bin"
	}
	
	// Return cache ID with extension for easier identification
	return fmt.Sprintf("att_%s%s", hash[:12], ext)
}