package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/prasanthmj/email/pkg/email"
	"github.com/prasanthmj/email/pkg/storage"
)

// handleListAccounts handles the list_accounts tool
func (h *Handler) handleListAccounts(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Handle case where no accounts are configured
	if len(h.config.Accounts) == 0 {
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{
				{
					Type: "text",
					Text: "No email accounts configured.\n\n" +
						"To configure accounts, set these environment variables:\n" +
						"  - ACCOUNT_{name}_EMAIL       (e.g., ACCOUNT_WORK_EMAIL=user@example.com)\n" +
						"  - ACCOUNT_{name}_PASSWORD    (e.g., ACCOUNT_WORK_PASSWORD=your_app_password)\n" +
						"  - DEFAULT_ACCOUNT_ID         (e.g., DEFAULT_ACCOUNT_ID=WORK)\n\n" +
						"For Gmail, use an App Password instead of your regular password.\n" +
						"Visit: https://myaccount.google.com/apppasswords\n\n" +
						"Example configuration:\n" +
						"  ACCOUNT_WORK_EMAIL=user@example.com\n" +
						"  ACCOUNT_WORK_PASSWORD=your_app_password\n" +
						"  ACCOUNT_WORK_PROVIDER=gmail\n" +
						"  DEFAULT_ACCOUNT_ID=WORK",
				},
			},
		}, nil
	}

	type AccountInfo struct {
		ID           string `json:"id"`
		EmailAddress string `json:"email"`
		Provider     string `json:"provider"`
		IsDefault    bool   `json:"is_default"`
	}

	accounts := make([]AccountInfo, 0, len(h.config.Accounts))
	for id, acct := range h.config.Accounts {
		accounts = append(accounts, AccountInfo{
			ID:           id,
			EmailAddress: acct.EmailAddress,
			Provider:     acct.Provider,
			IsDefault:    id == h.config.DefaultAccountID,
		})
	}

	// Sort by default first, then alphabetically
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].IsDefault != accounts[j].IsDefault {
			return accounts[i].IsDefault
		}
		return accounts[i].ID < accounts[j].ID
	})

	data, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// handleListFolders handles the list_folders tool
func (h *Handler) handleListFolders(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract account_id
	var accountID string
	if id, ok := args["account_id"].(string); ok {
		accountID = id
	}

	imapClient, err := h.getIMAPClient(accountID)
	if err != nil {
		return nil, err
	}

	folders, err := imapClient.ListFolders()
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	// Convert to JSON for response
	data, err := json.MarshalIndent(folders, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// handleFetchEmailHeaders handles the fetch_email_headers tool
func (h *Handler) handleFetchEmailHeaders(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract account_id
	var accountID string
	if id, ok := args["account_id"].(string); ok {
		accountID = id
	}

	opts := email.FetchOptions{
		Folder: "INBOX",
		Limit:  50,
	}

	// Parse folder
	if folder, ok := args["folder"].(string); ok && folder != "" {
		opts.Folder = folder
	}

	// Parse dates
	if sinceDate, ok := args["since_date"].(string); ok && sinceDate != "" {
		t, err := time.Parse("2006-01-02", sinceDate)
		if err != nil {
			return nil, fmt.Errorf("invalid since_date format (use YYYY-MM-DD): %w", err)
		}
		opts.SinceDate = t
	}

	if untilDate, ok := args["until_date"].(string); ok && untilDate != "" {
		t, err := time.Parse("2006-01-02", untilDate)
		if err != nil {
			return nil, fmt.Errorf("invalid until_date format (use YYYY-MM-DD): %w", err)
		}
		opts.UntilDate = t
	}

	// Parse filters
	if from, ok := args["from"].(string); ok {
		opts.From = from
	}

	if subject, ok := args["subject_contains"].(string); ok {
		opts.SubjectContains = subject
	}

	if unreadOnly, ok := args["unread_only"].(bool); ok {
		opts.UnreadOnly = unreadOnly
	}

	// Parse limit
	if limit, ok := args["limit"].(float64); ok {
		opts.Limit = int(limit)
	}

	// Fetch headers
	imapClient, err := h.getIMAPClient(accountID)
	if err != nil {
		return nil, err
	}

	headers, err := imapClient.FetchHeaders(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch email headers: %w", err)
	}

	// Convert to JSON for response
	data, err := json.MarshalIndent(headers, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// handleFetchEmail handles the fetch_email tool (enhanced version)
// Fetches email to cache and returns metadata with a text preview.
// Body content should be read using read_email_body tool.
func (h *Handler) handleFetchEmail(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract account_id
	var accountID string
	if id, ok := args["account_id"].(string); ok {
		accountID = id
	}
	accountID = h.resolveAccountID(accountID)

	messageID, ok := args["message_id"].(string)
	if !ok || messageID == "" {
		return nil, fmt.Errorf("message_id parameter is required")
	}

	// Extract optional parameters
	previewLength := 500
	if pl, ok := args["preview_length"].(float64); ok {
		previewLength = int(pl)
	}

	// Get email cache
	emailCache, err := h.getEmailCache(accountID)
	if err != nil {
		return nil, err
	}

	// Check if already cached
	if !emailCache.IsCached(messageID) {
		// Not in cache, fetch from server
		imapClient, err := h.getIMAPClient(accountID)
		if err != nil {
			return nil, err
		}

		emailMsg, err := imapClient.FetchEmail(messageID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch email: %w", err)
		}

		// Save to cache with separate body files
		if _, err := emailCache.SaveEmail(emailMsg, accountID); err != nil {
			return nil, fmt.Errorf("failed to cache email: %w", err)
		}
	}

	// Get cache info (metadata + preview)
	cacheInfo, err := emailCache.GetCacheInfo(messageID, previewLength)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache info: %w", err)
	}

	// Convert to JSON for response
	data, err := json.MarshalIndent(cacheInfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// handleReadEmailBody handles the read_email_body tool
// Reads email body content from cache with pagination support.
// Default format is "text" which returns plain text (or HTML converted to text).
func (h *Handler) handleReadEmailBody(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract account_id
	var accountID string
	if id, ok := args["account_id"].(string); ok {
		accountID = id
	}
	accountID = h.resolveAccountID(accountID)

	messageID, ok := args["message_id"].(string)
	if !ok || messageID == "" {
		return nil, fmt.Errorf("message_id parameter is required")
	}

	// Extract optional parameters
	format := "text" // default to text
	if f, ok := args["format"].(string); ok && f != "" {
		format = f
	}

	// Validate format
	if format != "text" && format != "raw_html" {
		return nil, fmt.Errorf("invalid format: %s (must be 'text' or 'raw_html')", format)
	}

	var offset int64 = 0
	if o, ok := args["offset"].(float64); ok {
		offset = int64(o)
	}

	var limit int64 = 10000 // default 10k characters
	if l, ok := args["limit"].(float64); ok {
		limit = int64(l)
	}

	// Get email cache
	emailCache, err := h.getEmailCache(accountID)
	if err != nil {
		return nil, err
	}

	// Check if email is cached
	if !emailCache.IsCached(messageID) {
		return nil, fmt.Errorf("email not in cache. Call fetch_email first with message_id: %s", messageID)
	}

	// Read body content
	result, err := emailCache.ReadBody(messageID, format, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read email body: %w", err)
	}

	// Convert to JSON for response
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// getEmailCache returns the email cache for the account
func (h *Handler) getEmailCache(accountID string) (*storage.EmailCache, error) {
	clients, acctCfg, err := h.getAccountClients(accountID)
	if err != nil {
		return nil, err
	}

	if clients.emailCache == nil {
		// Get the files root from drafts dir (remove /drafts suffix)
		filesRoot := acctCfg.DraftsDir[:len(acctCfg.DraftsDir)-len("/drafts")]
		clients.emailCache = storage.NewEmailCache(filesRoot, h.config.CacheMaxSize)
	}
	return clients.emailCache, nil
}

// handleSendEmail handles the send_email tool
func (h *Handler) handleSendEmail(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract account_id
	var accountID string
	if id, ok := args["account_id"].(string); ok {
		accountID = id
	}

	opts := email.SendOptions{}

	// Parse recipients
	if to, ok := args["to"].([]interface{}); ok {
		for _, t := range to {
			if addr, ok := t.(string); ok {
				opts.To = append(opts.To, addr)
			}
		}
	} else if to, ok := args["to"].([]string); ok {
		opts.To = to
	}
	if len(opts.To) == 0 {
		return nil, fmt.Errorf("at least one 'to' recipient is required")
	}

	if cc, ok := args["cc"].([]interface{}); ok {
		for _, c := range cc {
			if addr, ok := c.(string); ok {
				opts.CC = append(opts.CC, addr)
			}
		}
	}

	if bcc, ok := args["bcc"].([]interface{}); ok {
		for _, b := range bcc {
			if addr, ok := b.(string); ok {
				opts.BCC = append(opts.BCC, addr)
			}
		}
	}

	// Parse subject
	if subject, ok := args["subject"].(string); ok {
		opts.Subject = subject
	}
	if opts.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}

	// Parse body
	if body, ok := args["body"].(string); ok {
		opts.Body = body
	}
	if htmlBody, ok := args["html_body"].(string); ok {
		opts.HTMLBody = htmlBody
	}
	if opts.Body == "" && opts.HTMLBody == "" {
		return nil, fmt.Errorf("either 'body' or 'html_body' is required")
	}

	// Parse attachments
	if attachments, ok := args["attachments"].([]interface{}); ok {
		for _, a := range attachments {
			if cacheID, ok := a.(string); ok {
				opts.Attachments = append(opts.Attachments, cacheID)
			}
		}
	}

	// Parse threading parameters
	if replyTo, ok := args["reply_to_message_id"].(string); ok {
		opts.ReplyToMessageID = replyTo
	}
	if references, ok := args["references"].([]interface{}); ok {
		for _, r := range references {
			if ref, ok := r.(string); ok {
				opts.References = append(opts.References, ref)
			}
		}
	}

	// Send the email
	smtpClient, err := h.getSMTPClient(accountID)
	if err != nil {
		return nil, err
	}

	if err := smtpClient.SendEmail(opts); err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Email sent successfully to %v", opts.To),
			},
		},
	}, nil
}

// handleFetchEmailAttachment handles the fetch_email_attachment tool
func (h *Handler) handleFetchEmailAttachment(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract account_id
	var accountID string
	if id, ok := args["account_id"].(string); ok {
		accountID = id
	}

	messageID, ok := args["message_id"].(string)
	if !ok || messageID == "" {
		return nil, fmt.Errorf("message_id parameter is required")
	}

	var attachmentNames []string
	if names, ok := args["attachment_names"].([]interface{}); ok {
		for _, n := range names {
			if name, ok := n.(string); ok {
				attachmentNames = append(attachmentNames, name)
			}
		}
	}

	fetchAll := false
	if fa, ok := args["fetch_all"].(bool); ok {
		fetchAll = fa
	}

	// Fetch attachments
	attFetcher, err := h.getAttachmentFetcher(accountID)
	if err != nil {
		return nil, err
	}
	
	results, err := attFetcher.FetchAttachments(messageID, attachmentNames, fetchAll)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attachments: %w", err)
	}

	// Format response
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}