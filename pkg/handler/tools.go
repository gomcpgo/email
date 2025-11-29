package handler

import (
	"encoding/json"

	"github.com/gomcpgo/mcp/pkg/protocol"
)

// GetTools returns the list of available tools
func GetTools() []protocol.Tool {
	return []protocol.Tool{
		{
			Name:        "list_accounts",
			Description: "List all configured email accounts with their IDs, email addresses, and which is the default account. Use this to discover available accounts before using account_id parameter in other tools.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {},
				"required": []
			}`),
		},
		{
			Name:        "list_folders",
			Description: "List all available email folders/labels with message counts. Use account_id parameter to specify which email account to query (call list_accounts first to see available accounts).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					}
				},
				"required": []
			}`),
		},
		{
			Name:        "fetch_email_headers",
			Description: "Fetch email headers (metadata) without bodies. Use this to list emails before fetching full content. Be mindful of the limit parameter as fetching many emails uses memory. Use account_id parameter to specify which email account to query (call list_accounts first to see available accounts).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					},
					"folder": {
						"type": "string",
						"description": "Email folder to fetch from (e.g., 'INBOX', 'Sent'). Default: INBOX"
					},
					"since_date": {
						"type": "string",
						"description": "Fetch emails since this date (ISO format: 2024-01-20)"
					},
					"until_date": {
						"type": "string",
						"description": "Fetch emails until this date (ISO format: 2024-01-27)"
					},
					"from": {
						"type": "string",
						"description": "Filter by sender email address"
					},
					"subject_contains": {
						"type": "string",
						"description": "Filter by subject containing this text"
					},
					"unread_only": {
						"type": "boolean",
						"description": "Only fetch unread emails. Default: false"
					},
					"limit": {
						"type": "integer",
						"description": "Maximum number of emails to fetch. Be mindful of memory usage. Default: 50"
					}
				},
				"required": []
			}`),
		},
		{
			Name:        "fetch_email",
			Description: "Fetch an email and cache it locally. Returns email metadata (headers, subject, from, to, date, attachments) and a text preview. The full body content is cached and can be read in chunks using read_email_body. This design prevents context overflow from large emails.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					},
					"message_id": {
						"type": "string",
						"description": "The Message-ID header value (e.g., '<CADsK8=example@mail.gmail.com>')"
					},
					"preview_length": {
						"type": "integer",
						"description": "Number of characters to include in the text preview. Default: 500"
					}
				},
				"required": ["message_id"]
			}`),
		},
		{
			Name:        "read_email_body",
			Description: "Read email body content from cache with pagination. Call fetch_email first to cache the email. Default format is 'text' which returns plain text (or HTML converted to text if no plain text exists). Use offset and limit for pagination of large emails.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					},
					"message_id": {
						"type": "string",
						"description": "The Message-ID of the email (must have been fetched first using fetch_email)"
					},
					"format": {
						"type": "string",
						"enum": ["text", "raw_html"],
						"description": "Content format: 'text' (default) returns plain text or HTML converted to text; 'raw_html' returns raw HTML"
					},
					"offset": {
						"type": "integer",
						"description": "Character position to start reading from. Default: 0"
					},
					"limit": {
						"type": "integer",
						"description": "Maximum characters to return. Default: 10000"
					}
				},
				"required": ["message_id"]
			}`),
		},
		{
			Name:        "send_email",
			Description: "Send an email. Properly sets threading headers for replies. Use account_id parameter to specify which email account to send from (call list_accounts first to see available accounts).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					},
					"to": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Recipient email addresses"
					},
					"cc": {
						"type": "array",
						"items": {"type": "string"},
						"description": "CC recipient email addresses"
					},
					"bcc": {
						"type": "array",
						"items": {"type": "string"},
						"description": "BCC recipient email addresses (hidden from other recipients)"
					},
					"subject": {
						"type": "string",
						"description": "Email subject line"
					},
					"body": {
						"type": "string",
						"description": "Plain text email body"
					},
					"html_body": {
						"type": "string",
						"description": "HTML email body (optional)"
					},
					"attachments": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Cache IDs of attachments to include (from fetch_email_attachment)"
					},
					"reply_to_message_id": {
						"type": "string",
						"description": "Message-ID of email being replied to (for threading)"
					},
					"references": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Message-IDs for threading chain"
					}
				},
				"required": ["to", "subject"]
			}`),
		},
		{
			Name:        "fetch_email_attachment",
			Description: "Fetch attachments from an email. Files are saved to cache for use with send_email. Maximum attachment size: 25MB. Use account_id parameter to specify which email account to query (call list_accounts first to see available accounts).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					},
					"message_id": {
						"type": "string",
						"description": "The Message-ID header value of the email"
					},
					"attachment_names": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Specific attachment filenames to fetch (e.g., ['report.pdf', 'image.png'])"
					},
					"fetch_all": {
						"type": "boolean",
						"description": "Fetch all attachments from the email. Default: false"
					}
				},
				"required": ["message_id"]
			}`),
		},
		{
			Name:        "create_draft",
			Description: "Create a new email draft. Save an email composition for later sending or editing. Use account_id parameter to specify which email account to use (call list_accounts first to see available accounts).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					},
					"to": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Recipient email addresses"
					},
					"cc": {
						"type": "array",
						"items": {"type": "string"},
						"description": "CC recipient email addresses"
					},
					"bcc": {
						"type": "array",
						"items": {"type": "string"},
						"description": "BCC recipient email addresses (hidden from other recipients)"
					},
					"subject": {
						"type": "string",
						"description": "Email subject line"
					},
					"body": {
						"type": "string",
						"description": "Plain text email body"
					},
					"html_body": {
						"type": "string",
						"description": "HTML email body (optional)"
					},
					"attachments": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Cache IDs of attachments to include"
					},
					"reply_to_message_id": {
						"type": "string",
						"description": "Message-ID of email being replied to (for threading)"
					},
					"references": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Message-IDs for threading chain"
					}
				},
				"required": []
			}`),
		},
		{
			Name:        "list_drafts",
			Description: "List all saved email drafts with their summaries. Use account_id parameter to specify which email account to query (call list_accounts first to see available accounts).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					}
				},
				"required": []
			}`),
		},
		{
			Name:        "get_draft",
			Description: "Retrieve a specific draft by its ID to view or edit. Use account_id parameter to specify which email account to query (call list_accounts first to see available accounts).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					},
					"draft_id": {
						"type": "string",
						"description": "The ID of the draft to retrieve"
					}
				},
				"required": ["draft_id"]
			}`),
		},
		{
			Name:        "update_draft",
			Description: "Update an existing draft. Only provided fields will be updated. Use account_id parameter to specify which email account to use (call list_accounts first to see available accounts).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					},
					"draft_id": {
						"type": "string",
						"description": "The ID of the draft to update"
					},
					"to": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Updated recipient email addresses"
					},
					"cc": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Updated CC recipients"
					},
					"bcc": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Updated BCC recipients"
					},
					"subject": {
						"type": "string",
						"description": "Updated subject line"
					},
					"body": {
						"type": "string",
						"description": "Updated plain text body"
					},
					"html_body": {
						"type": "string",
						"description": "Updated HTML body"
					},
					"attachments": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Updated attachment cache IDs"
					}
				},
				"required": ["draft_id"]
			}`),
		},
		{
			Name:        "send_draft",
			Description: "Send a draft email and remove it from drafts storage. Use account_id parameter to specify which email account to send from (call list_accounts first to see available accounts).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					},
					"draft_id": {
						"type": "string",
						"description": "The ID of the draft to send"
					}
				},
				"required": ["draft_id"]
			}`),
		},
		{
			Name:        "delete_draft",
			Description: "Delete a draft without sending it. Use account_id parameter to specify which email account to use (call list_accounts first to see available accounts).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					},
					"draft_id": {
						"type": "string",
						"description": "The ID of the draft to delete"
					}
				},
				"required": ["draft_id"]
			}`),
		},
		{
			Name:        "send_all_drafts",
			Description: "Send all drafts with a configurable delay between each email to avoid rate limits. Use account_id parameter to specify which email account to send from (call list_accounts first to see available accounts).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"account_id": {
						"type": "string",
						"description": "Account ID to use. If not specified, uses the default account from DEFAULT_ACCOUNT_ID"
					},
					"delay_seconds": {
						"type": "integer",
						"description": "Seconds to wait between sending each email (2-60). Default: 5"
					},
					"dry_run": {
						"type": "boolean",
						"description": "If true, simulate sending without actually sending. Default: false"
					},
					"stop_on_error": {
						"type": "boolean",
						"description": "If true, stop sending if any email fails. Default: false"
					}
				},
				"required": []
			}`),
		},
	}
}