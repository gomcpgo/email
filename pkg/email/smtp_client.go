package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"path/filepath"
	"strings"

	"github.com/jordan-wright/email"
	"github.com/prasanthmj/email/pkg/config"
)

// SMTPClient handles SMTP operations
type SMTPClient struct {
	config *config.AccountConfig
}

// NewSMTPClient creates a new SMTP client
func NewSMTPClient(cfg *config.AccountConfig) *SMTPClient {
	return &SMTPClient{
		config: cfg,
	}
}

// SendEmail sends an email with the given options
func (sc *SMTPClient) SendEmail(opts SendOptions) error {
	e := email.NewEmail()
	
	// Set from address
	e.From = sc.config.EmailAddress
	
	// Set recipients
	if len(opts.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	e.To = opts.To
	
	if len(opts.CC) > 0 {
		e.Cc = opts.CC
	}
	
	if len(opts.BCC) > 0 {
		e.Bcc = opts.BCC
	}
	
	// Set subject
	if opts.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	e.Subject = opts.Subject
	
	// Set body
	if opts.Body != "" {
		e.Text = []byte(opts.Body)
	}
	
	if opts.HTMLBody != "" {
		e.HTML = []byte(opts.HTMLBody)
	}
	
	// If neither body is provided
	if opts.Body == "" && opts.HTMLBody == "" {
		return fmt.Errorf("email body is required")
	}
	
	// Set threading headers if this is a reply
	if opts.ReplyToMessageID != "" {
		e.Headers.Set("In-Reply-To", opts.ReplyToMessageID)
		
		// Build References header
		refs := opts.References
		if !contains(refs, opts.ReplyToMessageID) {
			refs = append(refs, opts.ReplyToMessageID)
		}
		if len(refs) > 0 {
			e.Headers.Set("References", strings.Join(refs, " "))
		}
	}
	
	// Add attachments from cache
	for _, cacheID := range opts.Attachments {
		attachmentPath := filepath.Join(sc.config.AttachmentDir, cacheID)
		_, err := e.AttachFile(attachmentPath)
		if err != nil {
			return fmt.Errorf("failed to attach file %s: %w", cacheID, err)
		}
	}
	
	// Send the email
	addr := fmt.Sprintf("%s:%d", sc.config.SMTPServer, sc.config.SMTPPort)
	
	// Create auth
	auth := smtp.PlainAuth("", sc.config.EmailAddress, sc.config.EmailPassword, sc.config.SMTPServer)
	
	// Send with TLS
	err := e.SendWithStartTLS(addr, auth, &tls.Config{
		ServerName: sc.config.SMTPServer,
	})
	
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	return nil
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}