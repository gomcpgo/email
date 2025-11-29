# Email MCP Server

A Model Context Protocol (MCP) server for email operations via IMAP and SMTP. Supports Gmail, Outlook, and other IMAP/SMTP servers with multi-account support.

## Features

- **Multi-account support** - Manage multiple email accounts simultaneously
- **List email folders** - Enumerate all available folders/labels
- **Fetch email headers** - Get email metadata without downloading full content
- **Fetch and cache emails** - Download emails with smart caching to prevent context overflow
- **Read email body in chunks** - Pagination support for large emails
- **HTML to text conversion** - Automatic conversion for LLM-friendly output
- **Send emails** - Send emails with proper threading support for replies
- **Fetch attachments** - Download and cache email attachments
- **Draft management** - Create, edit, and manage email drafts

## Multi-Account Support

The server supports managing multiple email accounts. Each account has its own:
- Isolated storage (drafts, cache, attachments)
- Independent IMAP/SMTP configurations
- Provider-specific auto-configuration (Gmail, Outlook)

All MCP tools accept an optional `account_id` parameter. If not specified, the default account (configured via `DEFAULT_ACCOUNT_ID`) is used.

## Configuration

### Multi-Account Configuration

Copy `.env.example` to `.env` and configure your accounts:

```bash
# Set the default account
DEFAULT_ACCOUNT_ID=work

# Account 1: Work Email (Gmail)
ACCOUNT_work_EMAIL=work@company.com
ACCOUNT_work_PASSWORD=your_app_password_here
ACCOUNT_work_PROVIDER=gmail

# Account 2: Personal Email (Gmail)
ACCOUNT_personal_EMAIL=me@gmail.com
ACCOUNT_personal_PASSWORD=your_app_password_here
ACCOUNT_personal_PROVIDER=gmail

# Account 3: Custom Email Server
ACCOUNT_custom_EMAIL=user@custom-domain.com
ACCOUNT_custom_PASSWORD=your_password_here
ACCOUNT_custom_PROVIDER=custom
ACCOUNT_custom_IMAP_SERVER=mail.custom-domain.com
ACCOUNT_custom_IMAP_PORT=993
ACCOUNT_custom_SMTP_SERVER=mail.custom-domain.com
ACCOUNT_custom_SMTP_PORT=587

# Global storage settings
FILES_ROOT=/tmp/email-mcp              # Root directory for all accounts
EMAIL_CACHE_MAX_SIZE=10485760          # 10MB cache limit per account
EMAIL_MAX_ATTACHMENT_SIZE=26214400     # 25MB max attachment size
```

### Account Naming

- Account IDs can be any alphanumeric string (e.g., `work`, `personal`, `client1`)
- Use the pattern `ACCOUNT_{account_id}_{SETTING}` for all account-specific settings
- Each account's data is stored in `FILES_ROOT/{account_id}/`

### Gmail Setup

1. Enable 2-factor authentication
2. Generate an app password: https://myaccount.google.com/apppasswords
3. Use the app password as `EMAIL_APP_PASSWORD`

### Outlook Setup

1. Enable 2-factor authentication
2. Generate an app password: https://account.microsoft.com/security
3. Use the app password as `EMAIL_APP_PASSWORD`

## Installation

```bash
# Install dependencies
./run.sh install

# Build the server
./run.sh build
```

## Usage

### Run as MCP Server

```bash
./run.sh run
```

### Terminal Mode (Testing)

```bash
# List all email folders
./run.sh folders

# Fetch email headers from last 7 days
./run.sh fetch

# Fetch email headers since specific date
./run.sh fetch 2024-01-20

# Fetch a complete email
./run.sh email '<CADsK8=example@mail.gmail.com>'

# Send a test email
./run.sh send-test

# Fetch attachments from an email
./run.sh attachment '<CADsK8=example@mail.gmail.com>'

# Show cache statistics
./run.sh cache-info

# Clear cache
./run.sh clear-cache
```

## MCP Tools

**Note:** All tools accept an optional `account_id` parameter to specify which account to use. If omitted, the default account (configured via `DEFAULT_ACCOUNT_ID`) is used.

### list_accounts
Lists all configured email accounts with their IDs and which is the default.

```json
{}
```

### list_folders
Lists all available email folders with message counts.

```json
{
  "account_id": "work"  // Optional: defaults to DEFAULT_ACCOUNT_ID
}
```

### fetch_email_headers
Fetches email headers (metadata) without downloading full content. Use this to list/search emails before fetching full content.

```json
{
  "folder": "INBOX",
  "since_date": "2024-01-20",
  "until_date": "2024-01-27",
  "from": "sender@example.com",
  "subject_contains": "newsletter",
  "unread_only": false,
  "limit": 50
}
```

### fetch_email
Fetches an email and caches it locally. Returns email metadata (headers, subject, from, to, date, attachments) and a text preview. The full body content is cached and can be read in chunks using `read_email_body`.

This design prevents context overflow from large emails - the LLM receives metadata + preview, then decides whether to read the full body.

```json
{
  "message_id": "<CADsK8=example@mail.gmail.com>",
  "preview_length": 1000  // Optional: characters in preview (default: 1000)
}
```

**Response:**
```json
{
  "message_id": "<CADsK8=example@mail.gmail.com>",
  "from": "sender@example.com",
  "to": ["recipient@example.com"],
  "subject": "Email subject",
  "date": "2024-01-20T10:30:00Z",
  "attachments": [
    {"filename": "report.pdf", "size": 245000}
  ],
  "body": {
    "text_size": 5000,
    "html_size": 15000,
    "has_text": true,
    "has_html": true,
    "preview": "First 1000 characters of the email body..."
  }
}
```

### read_email_body
Reads email body content from cache with pagination support. Call `fetch_email` first to cache the email.

Default format is `text` which returns plain text. If the email only has HTML, it's automatically converted to text.

```json
{
  "message_id": "<CADsK8=example@mail.gmail.com>",
  "format": "text",      // Optional: "text" (default) or "raw_html"
  "offset": 0,           // Optional: character position to start (default: 0)
  "limit": 10000         // Optional: max characters to return (default: 10000)
}
```

**Response:**
```json
{
  "content": "The email body content...",
  "format": "text",
  "source": "text_body",  // or "html_converted" if HTML was converted
  "total_size": 5000,
  "offset": 0,
  "limit": 10000,
  "remaining": 0,
  "is_complete": true
}
```

**Pagination example for large emails:**
```json
// First chunk
{"message_id": "...", "offset": 0, "limit": 10000}
// Response: remaining: 15000, is_complete: false

// Second chunk  
{"message_id": "...", "offset": 10000, "limit": 10000}
// Response: remaining: 5000, is_complete: false

// Final chunk
{"message_id": "...", "offset": 20000, "limit": 10000}
// Response: remaining: 0, is_complete: true
```

### send_email
Sends an email with optional attachments and threading support.

```json
{
  "to": ["recipient@example.com"],
  "cc": ["cc@example.com"],
  "bcc": ["bcc@example.com"],
  "subject": "Email subject",
  "body": "Plain text body",
  "html_body": "<html>...</html>",
  "attachments": ["cache_id_1"],
  "reply_to_message_id": "<original@mail.com>",
  "references": ["<ref1@mail.com>"]
}
```

### fetch_email_attachment
Downloads attachments from an email to cache.

```json
{
  "message_id": "<CADsK8=example@mail.gmail.com>",
  "attachment_names": ["report.pdf", "image.png"],
  "fetch_all": false
}
```

### Draft Management Tools

- **create_draft** - Create a new email draft
- **list_drafts** - List all saved drafts
- **get_draft** - Retrieve a specific draft
- **update_draft** - Update an existing draft
- **send_draft** - Send a draft and remove it
- **delete_draft** - Delete a draft without sending
- **send_all_drafts** - Send all drafts with configurable delay

## Cache Management

The server caches emails and attachments for performance:

- **Location**: `$FILES_ROOT/{account_id}/cache/emails/`
- **Max size**: 10MB per account (configurable)
- **Expiry**: 96 hours (4 days)
- **Eviction**: Oldest entries first

### Cache Structure

Each cached email is stored in its own directory:
```
cache/emails/{cache_id}/
├── metadata.yaml      # Headers, sizes, attachment info
├── body_text.txt      # Plain text body (if available)
├── body_html.txt      # HTML body (if available)
└── body_converted.txt # HTML converted to text (cached on first request)
```

This structure allows:
- Reading body content in chunks without loading entire file
- Caching HTML-to-text conversion results
- Efficient pagination for large emails

## Performance & Token Optimization

The email server is designed to minimize LLM token usage:

### Two-Phase Email Reading

1. **fetch_email** - Returns metadata + preview (~1-2KB)
2. **read_email_body** - Returns full content in chunks (if needed)

This prevents large emails (50KB+) from overwhelming the context window.

### Response Size Comparison

| Operation | Typical Size | Use Case |
|-----------|--------------|----------|
| Email Headers | ~400 chars/email | Listing, searching |
| fetch_email | ~1-2KB | Metadata + preview |
| read_email_body | Up to 10KB/chunk | Full content when needed |

### Recommendations

- ✅ Use `fetch_email_headers` for listing/searching
- ✅ Use `fetch_email` to get metadata and preview
- ✅ Only call `read_email_body` when full content is needed
- ✅ Use pagination (`offset`/`limit`) for very large emails
- ⚠️ Avoid fetching many full emails in one request

## Testing

```bash
# Run unit tests
./run.sh test

# Test with your email account
export EMAIL_ADDRESS=test@gmail.com
export EMAIL_APP_PASSWORD=xxxx-xxxx-xxxx
./run.sh folders
```

## Security Notes

- App passwords are used instead of regular passwords
- Passwords are never logged or exposed in error messages
- BCC recipients are properly hidden
- Cache files are stored with 0644 permissions

## Troubleshooting

### Authentication Failed
- Verify app password is correct
- Check 2-factor authentication is enabled
- For Gmail, ensure "less secure apps" is not blocking access

### Folder Not Found
- Use `list_folders` to see exact folder names
- Gmail uses `[Gmail]/Sent Mail` instead of `Sent`
- Outlook uses `Sent Items` instead of `Sent`

### Cache Issues
- Run `./run.sh clear-cache` to reset
- Check `FILES_ROOT` directory permissions
- Verify disk space available

### Email Not in Cache
- Call `fetch_email` first before `read_email_body`
- Cache expires after 96 hours - refetch if needed

## License

MIT
