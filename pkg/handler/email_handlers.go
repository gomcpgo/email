package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/prasanthmj/email/pkg/email"
)

// handleListFolders handles the list_folders tool
func (h *Handler) handleListFolders(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	imapClient, err := h.getIMAPClient()
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
	imapClient, err := h.getIMAPClient()
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

// handleFetchEmail handles the fetch_email tool
func (h *Handler) handleFetchEmail(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	messageID, ok := args["message_id"].(string)
	if !ok || messageID == "" {
		return nil, fmt.Errorf("message_id parameter is required")
	}

	// Try to load from cache first
	cachedEmail, err := h.storage.LoadEmail(messageID)
	if err == nil {
		// Found in cache
		data, err := json.MarshalIndent(cachedEmail, "", "  ")
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

	// Not in cache, fetch from server
	imapClient, err := h.getIMAPClient()
	if err != nil {
		return nil, err
	}
	
	emailMsg, err := imapClient.FetchEmail(messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch email: %w", err)
	}

	// Save to cache
	if err := h.storage.SaveEmail(emailMsg); err != nil {
		// Log error but continue
		fmt.Printf("Failed to cache email: %v\n", err)
	}

	// Convert to JSON for response
	data, err := json.MarshalIndent(emailMsg, "", "  ")
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

// handleSendEmail handles the send_email tool
func (h *Handler) handleSendEmail(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
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
	smtpClient, err := h.getSMTPClient()
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
	attFetcher, err := h.getAttachmentFetcher()
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