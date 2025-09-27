package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/prasanthmj/email/pkg/email"
)

// handleCreateDraft handles the create_draft tool
func (h *Handler) handleCreateDraft(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
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

	// Parse subject and body
	if subject, ok := args["subject"].(string); ok {
		opts.Subject = subject
	}

	if body, ok := args["body"].(string); ok {
		opts.Body = body
	}

	if htmlBody, ok := args["html_body"].(string); ok {
		opts.HTMLBody = htmlBody
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

	// Save the draft
	draftID, err := h.storage.SaveDraft(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to save draft: %w", err)
	}

	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Draft saved with ID: %s", draftID),
			},
		},
	}, nil
}

// handleListDrafts handles the list_drafts tool
func (h *Handler) handleListDrafts(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	drafts, err := h.storage.ListDrafts()
	if err != nil {
		return nil, fmt.Errorf("failed to list drafts: %w", err)
	}

	// Convert to JSON for response
	data, err := json.MarshalIndent(drafts, "", "  ")
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

// handleGetDraft handles the get_draft tool
func (h *Handler) handleGetDraft(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	draftID, ok := args["draft_id"].(string)
	if !ok || draftID == "" {
		return nil, fmt.Errorf("draft_id parameter is required")
	}

	draft, err := h.storage.LoadDraft(draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to load draft: %w", err)
	}

	// Convert to JSON for response
	data, err := json.MarshalIndent(draft, "", "  ")
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

// handleUpdateDraft handles the update_draft tool
func (h *Handler) handleUpdateDraft(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	draftID, ok := args["draft_id"].(string)
	if !ok || draftID == "" {
		return nil, fmt.Errorf("draft_id parameter is required")
	}

	// Load existing draft first
	existingDraft, err := h.storage.LoadDraft(draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing draft: %w", err)
	}

	// Build updated SendOptions from existing draft
	opts := email.SendOptions{
		To:               existingDraft.To,
		CC:               existingDraft.CC,
		BCC:              existingDraft.BCC,
		Subject:          existingDraft.Subject,
		Body:             existingDraft.Body,
		HTMLBody:         existingDraft.HTMLBody,
		Attachments:      existingDraft.Attachments,
		ReplyToMessageID: existingDraft.ReplyToMessageID,
		References:       existingDraft.References,
	}

	// Apply updates
	if to, ok := args["to"].([]interface{}); ok {
		opts.To = nil
		for _, t := range to {
			if addr, ok := t.(string); ok {
				opts.To = append(opts.To, addr)
			}
		}
	} else if to, ok := args["to"].([]string); ok {
		opts.To = to
	}

	if cc, ok := args["cc"].([]interface{}); ok {
		opts.CC = nil
		for _, c := range cc {
			if addr, ok := c.(string); ok {
				opts.CC = append(opts.CC, addr)
			}
		}
	}

	if bcc, ok := args["bcc"].([]interface{}); ok {
		opts.BCC = nil
		for _, b := range bcc {
			if addr, ok := b.(string); ok {
				opts.BCC = append(opts.BCC, addr)
			}
		}
	}

	if subject, ok := args["subject"].(string); ok {
		opts.Subject = subject
	}

	if body, ok := args["body"].(string); ok {
		opts.Body = body
	}

	if htmlBody, ok := args["html_body"].(string); ok {
		opts.HTMLBody = htmlBody
	}

	if attachments, ok := args["attachments"].([]interface{}); ok {
		opts.Attachments = nil
		for _, a := range attachments {
			if cacheID, ok := a.(string); ok {
				opts.Attachments = append(opts.Attachments, cacheID)
			}
		}
	}

	// Delete old draft and save new one with same ID
	if err := h.storage.DeleteDraft(draftID); err != nil {
		return nil, fmt.Errorf("failed to delete old draft: %w", err)
	}

	newDraftID, err := h.storage.SaveDraft(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to save updated draft: %w", err)
	}

	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Draft updated successfully. New ID: %s", newDraftID),
			},
		},
	}, nil
}

// handleSendDraft handles the send_draft tool
func (h *Handler) handleSendDraft(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	draftID, ok := args["draft_id"].(string)
	if !ok || draftID == "" {
		return nil, fmt.Errorf("draft_id parameter is required")
	}

	// Load the draft
	draft, err := h.storage.LoadDraft(draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to load draft: %w", err)
	}

	// Convert draft to SendOptions
	opts := email.SendOptions{
		To:               draft.To,
		CC:               draft.CC,
		BCC:              draft.BCC,
		Subject:          draft.Subject,
		Body:             draft.Body,
		HTMLBody:         draft.HTMLBody,
		Attachments:      draft.Attachments,
		ReplyToMessageID: draft.ReplyToMessageID,
		References:       draft.References,
	}

	// Validate required fields
	if len(opts.To) == 0 {
		return nil, fmt.Errorf("draft has no recipients")
	}
	if opts.Subject == "" {
		return nil, fmt.Errorf("draft has no subject")
	}
	if opts.Body == "" && opts.HTMLBody == "" {
		return nil, fmt.Errorf("draft has no content")
	}

	// Send the email
	smtpClient, err := h.getSMTPClient()
	if err != nil {
		return nil, err
	}

	if err := smtpClient.SendEmail(opts); err != nil {
		return nil, fmt.Errorf("failed to send draft: %w", err)
	}

	// Delete the draft after successful send
	if err := h.storage.DeleteDraft(draftID); err != nil {
		// Log error but don't fail - email was sent successfully
		fmt.Printf("Warning: failed to delete draft after sending: %v\n", err)
	}

	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Draft sent successfully to %v and removed from drafts", opts.To),
			},
		},
	}, nil
}

// handleDeleteDraft handles the delete_draft tool
func (h *Handler) handleDeleteDraft(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	draftID, ok := args["draft_id"].(string)
	if !ok || draftID == "" {
		return nil, fmt.Errorf("draft_id parameter is required")
	}

	if err := h.storage.DeleteDraft(draftID); err != nil {
		return nil, fmt.Errorf("failed to delete draft: %w", err)
	}

	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Draft %s deleted successfully", draftID),
			},
		},
	}, nil
}

// handleSendAllDrafts handles the send_all_drafts tool
func (h *Handler) handleSendAllDrafts(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Parse parameters
	delaySeconds := 5
	if delay, ok := args["delay_seconds"].(float64); ok {
		delaySeconds = int(delay)
		if delaySeconds < 2 {
			delaySeconds = 2
		} else if delaySeconds > 60 {
			delaySeconds = 60
		}
	}

	dryRun := false
	if dr, ok := args["dry_run"].(bool); ok {
		dryRun = dr
	}

	stopOnError := false
	if soe, ok := args["stop_on_error"].(bool); ok {
		stopOnError = soe
	}

	// Get all drafts
	drafts, err := h.storage.ListDrafts()
	if err != nil {
		return nil, fmt.Errorf("failed to list drafts: %w", err)
	}

	if len(drafts) == 0 {
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{
				{
					Type: "text",
					Text: "No drafts to send",
				},
			},
		}, nil
	}

	// Prepare SMTP client if not dry run
	var smtpClient *email.SMTPClient
	if !dryRun {
		smtpClient, err = h.getSMTPClient()
		if err != nil {
			return nil, err
		}
	}

	// Send results tracking
	type sendResult struct {
		DraftID string `json:"draft_id"`
		Subject string `json:"subject"`
		To      []string `json:"to"`
		Status  string `json:"status"`
		Error   string `json:"error,omitempty"`
	}

	var results []sendResult
	successCount := 0
	failCount := 0

	for i, draftSummary := range drafts {
		// Load full draft
		draft, err := h.storage.LoadDraft(draftSummary.ID)
		if err != nil {
			result := sendResult{
				DraftID: draftSummary.ID,
				Subject: draftSummary.Subject,
				To:      draftSummary.To,
				Status:  "failed",
				Error:   fmt.Sprintf("failed to load draft: %v", err),
			}
			results = append(results, result)
			failCount++
			
			if stopOnError {
				break
			}
			continue
		}

		if !dryRun {
			// Convert to SendOptions
			opts := email.SendOptions{
				To:               draft.To,
				CC:               draft.CC,
				BCC:              draft.BCC,
				Subject:          draft.Subject,
				Body:             draft.Body,
				HTMLBody:         draft.HTMLBody,
				Attachments:      draft.Attachments,
				ReplyToMessageID: draft.ReplyToMessageID,
				References:       draft.References,
			}

			// Send the email
			if err := smtpClient.SendEmail(opts); err != nil {
				result := sendResult{
					DraftID: draft.ID,
					Subject: draft.Subject,
					To:      draft.To,
					Status:  "failed",
					Error:   fmt.Sprintf("send failed: %v", err),
				}
				results = append(results, result)
				failCount++
				
				if stopOnError {
					break
				}
			} else {
				// Success - delete the draft
				h.storage.DeleteDraft(draft.ID)
				
				result := sendResult{
					DraftID: draft.ID,
					Subject: draft.Subject,
					To:      draft.To,
					Status:  "sent",
				}
				results = append(results, result)
				successCount++
			}

			// Delay between sends (except for last one)
			if i < len(drafts)-1 {
				time.Sleep(time.Duration(delaySeconds) * time.Second)
			}
		} else {
			// Dry run - just simulate
			result := sendResult{
				DraftID: draft.ID,
				Subject: draft.Subject,
				To:      draft.To,
				Status:  "simulated",
			}
			results = append(results, result)
			successCount++
		}
	}

	// Prepare summary
	summary := map[string]interface{}{
		"total_drafts": len(drafts),
		"sent":         successCount,
		"failed":       failCount,
		"dry_run":      dryRun,
		"delay_seconds": delaySeconds,
		"results":      results,
	}

	data, err := json.MarshalIndent(summary, "", "  ")
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