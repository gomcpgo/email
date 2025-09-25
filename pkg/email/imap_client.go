package email

import (
	"fmt"
	"io"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/prasanthmj/email/pkg/config"
)

// IMAPClient handles IMAP operations
type IMAPClient struct {
	config *config.Config
}

// NewIMAPClient creates a new IMAP client
func NewIMAPClient(cfg *config.Config) *IMAPClient {
	return &IMAPClient{
		config: cfg,
	}
}

// connect establishes a connection to the IMAP server
func (ic *IMAPClient) connect() (*client.Client, error) {
	addr := fmt.Sprintf("%s:%d", ic.config.IMAPServer, ic.config.IMAPPort)
	
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to email server: %w", err)
	}
	
	// Set timeout
	c.Timeout = ic.config.Timeout
	
	// Login
	if err := c.Login(ic.config.EmailAddress, ic.config.EmailPassword); err != nil {
		c.Logout()
		return nil, fmt.Errorf("authentication failed")
	}
	
	return c, nil
}

// ListFolders returns all available folders
func (ic *IMAPClient) ListFolders() ([]Folder, error) {
	c, err := ic.connect()
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	var folders []Folder
	for m := range mailboxes {
		// Get folder status for counts
		mbox, err := c.Select(m.Name, true)
		if err == nil {
			folders = append(folders, Folder{
				Name:         m.Name,
				MessageCount: mbox.Messages,
				UnreadCount:  mbox.Unseen,
			})
		} else {
			// If we can't select, just add without counts
			folders = append(folders, Folder{
				Name: m.Name,
			})
		}
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	return folders, nil
}

// FetchHeaders fetches email headers based on options
func (ic *IMAPClient) FetchHeaders(opts FetchOptions) ([]EmailHeader, error) {
	c, err := ic.connect()
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	// Select folder
	folder := opts.Folder
	if folder == "" {
		folder = "INBOX"
	}
	
	mbox, err := c.Select(folder, true) // read-only
	if err != nil {
		return nil, fmt.Errorf("folder does not exist: %s", folder)
	}

	if mbox.Messages == 0 {
		return []EmailHeader{}, nil
	}

	// Build search criteria
	criteria := ic.buildSearchCriteria(opts)
	
	// Search for messages
	seqNums, err := c.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(seqNums) == 0 {
		return []EmailHeader{}, nil
	}

	// Apply limit
	if opts.Limit > 0 && len(seqNums) > opts.Limit {
		// Get the most recent messages
		seqNums = seqNums[len(seqNums)-opts.Limit:]
	}

	// Create sequence set
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(seqNums...)

	// Fetch message headers
	messages := make(chan *imap.Message, 10)
	section := &imap.BodySectionName{Peek: true}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchRFC822Size, section.FetchItem()}
	
	go func() {
		if err := c.Fetch(seqSet, items, messages); err != nil {
			// Log error but continue
		}
	}()

	var headers []EmailHeader
	for msg := range messages {
		if msg.Envelope == nil {
			continue
		}

		header := EmailHeader{
			MessageID:      msg.Envelope.MessageId,
			Folder:         folder,
			From:           formatAddress(msg.Envelope.From),
			To:             formatAddresses(msg.Envelope.To),
			CC:             formatAddresses(msg.Envelope.Cc),
			Subject:        msg.Envelope.Subject,
			Date:           msg.Envelope.Date,
			HasAttachments: hasAttachments(msg),
			IsUnread:       !hasFlag(msg, imap.SeenFlag),
			Size:           int64(msg.Size),
		}
		headers = append(headers, header)
	}

	return headers, nil
}

// FetchEmail fetches a complete email by Message-ID
func (ic *IMAPClient) FetchEmail(messageID string) (*Email, error) {
	c, err := ic.connect()
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	// Search all folders for the message
	email, err := ic.searchAndFetchEmail(c, messageID)
	if err != nil {
		return nil, err
	}

	return email, nil
}

// searchAndFetchEmail searches for and fetches an email from any folder
func (ic *IMAPClient) searchAndFetchEmail(c *client.Client, messageID string) (*Email, error) {
	// Try common folders first
	commonFolders := []string{"INBOX", "Sent", "[Gmail]/Sent Mail", "Sent Items", "[Gmail]/All Mail"}
	
	for _, folder := range commonFolders {
		email, err := ic.fetchEmailFromFolder(c, folder, messageID)
		if err == nil && email != nil {
			return email, nil
		}
	}

	// If not found in common folders, search all folders
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	for m := range mailboxes {
		email, err := ic.fetchEmailFromFolder(c, m.Name, messageID)
		if err == nil && email != nil {
			return email, nil
		}
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to search folders: %w", err)
	}

	return nil, fmt.Errorf("email not found: %s", messageID)
}

// fetchEmailFromFolder attempts to fetch an email from a specific folder
func (ic *IMAPClient) fetchEmailFromFolder(c *client.Client, folder, messageID string) (*Email, error) {
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
	seqSet.AddNum(seqNums[0]) // Take first match

	messages := make(chan *imap.Message, 1)
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchRFC822}
	
	go func() {
		if err := c.Fetch(seqSet, items, messages); err != nil {
			// Log error
		}
	}()

	msg := <-messages
	if msg == nil || msg.Envelope == nil {
		return nil, fmt.Errorf("failed to fetch message")
	}

	// Parse the message body
	var body string
	var htmlBody string
	var attachments []Attachment
	var inReplyTo string
	var references []string

	r := msg.GetBody(&imap.BodySectionName{})
	if r != nil {
		mr, err := mail.CreateReader(r)
		if err == nil {
			// Extract headers
			header := mr.Header
			if refs, err := header.AddressList("References"); err == nil {
				for _, ref := range refs {
					references = append(references, ref.Address)
				}
			}
			if irt, err := header.Text("In-Reply-To"); err == nil {
				inReplyTo = irt
			}

			// Extract body and attachments
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					break
				}

				switch h := p.Header.(type) {
				case *mail.InlineHeader:
					// This is the message body
					b, _ := io.ReadAll(p.Body)
					ct, _, _ := h.ContentType()
					if strings.Contains(ct, "text/html") {
						htmlBody = string(b)
					} else if strings.Contains(ct, "text/plain") {
						body = string(b)
					}
				case *mail.AttachmentHeader:
					// This is an attachment
					filename, _ := h.Filename()
					contentType, _, _ := h.ContentType()
					// Get size by reading (we won't store the content here)
					b, _ := io.ReadAll(p.Body)
					attachments = append(attachments, Attachment{
						Filename:    filename,
						Size:        int64(len(b)),
						ContentType: contentType,
					})
				}
			}
		}
	}

	email := &Email{
		MessageID:   messageID,
		Folder:      folder,
		From:        formatAddress(msg.Envelope.From),
		To:          formatAddresses(msg.Envelope.To),
		CC:          formatAddresses(msg.Envelope.Cc),
		BCC:         formatAddresses(msg.Envelope.Bcc),
		Subject:     msg.Envelope.Subject,
		Date:        msg.Envelope.Date,
		Body:        body,
		HTMLBody:    htmlBody,
		Attachments: attachments,
		InReplyTo:   inReplyTo,
		References:  references,
	}

	return email, nil
}

// buildSearchCriteria builds IMAP search criteria from options
func (ic *IMAPClient) buildSearchCriteria(opts FetchOptions) *imap.SearchCriteria {
	criteria := imap.NewSearchCriteria()
	
	if !opts.SinceDate.IsZero() {
		criteria.Since = opts.SinceDate
	}
	
	if !opts.UntilDate.IsZero() {
		criteria.Before = opts.UntilDate.AddDate(0, 0, 1) // Add one day for inclusive search
	}
	
	if opts.From != "" {
		criteria.Header.Set("From", opts.From)
	}
	
	if opts.SubjectContains != "" {
		criteria.Header.Set("Subject", opts.SubjectContains)
	}
	
	if opts.UnreadOnly {
		criteria.WithoutFlags = []string{imap.SeenFlag}
	}
	
	return criteria
}

// Helper functions

func formatAddress(addrs []*imap.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	addr := addrs[0]
	if addr.PersonalName != "" {
		return fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName)
	}
	return fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
}

func formatAddresses(addrs []*imap.Address) []string {
	var result []string
	for _, addr := range addrs {
		if addr.PersonalName != "" {
			result = append(result, fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName))
		} else {
			result = append(result, fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName))
		}
	}
	return result
}

func hasAttachments(msg *imap.Message) bool {
	if msg.BodyStructure == nil {
		return false
	}
	for _, part := range msg.BodyStructure.Parts {
		if part.Disposition == "attachment" {
			return true
		}
	}
	return false
}

func hasFlag(msg *imap.Message, flag string) bool {
	for _, f := range msg.Flags {
		if f == flag {
			return true
		}
	}
	return false
}