# Email MCP Server

A Model Context Protocol (MCP) server for email operations via IMAP and SMTP. Supports Gmail, Outlook, and other IMAP/SMTP servers.

## Features

- **List email folders** - Enumerate all available folders/labels
- **Fetch email headers** - Get email metadata without downloading full content
- **Fetch complete emails** - Download full email with body and attachments
- **Send emails** - Send emails with proper threading support for replies
- **Fetch attachments** - Download and cache email attachments

## Configuration

Copy `.env.example` to `.env` and configure your email settings:

```bash
# Email account settings
EMAIL_ADDRESS=your-email@gmail.com
EMAIL_APP_PASSWORD=your-app-password
EMAIL_PROVIDER=gmail              # gmail, outlook, or custom

# Storage settings
FILES_ROOT=/path/to/storage       # Root for drafts and cache
EMAIL_CACHE_MAX_SIZE=10485760     # 10MB cache limit
EMAIL_MAX_ATTACHMENT_SIZE=26214400 # 25MB max attachment
```

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

### list_folders
Lists all available email folders with message counts.

```json
{}
```

### fetch_email_headers
Fetches email headers without downloading full content.

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
Fetches a complete email with body and attachments.

```json
{
  "message_id": "<CADsK8=example@mail.gmail.com>"
}
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

## Cache Management

The server caches emails and attachments for performance:

- **Location**: `$FILES_ROOT/cache/`
- **Max size**: 10MB (configurable)
- **Expiry**: 1 day
- **Eviction**: Oldest entries first

Cached items:
- Email bodies (YAML format)
- Attachments (original format)

## Performance & Response Sizes

Understanding response sizes helps optimize LLM token usage:

### Response Size Comparison
- **Email Headers**: ~430 chars/email (metadata only)
- **Full Emails**: 7-80KB+ per email (13-200x larger)

### Commands for Size Analysis
```bash
# Compare header vs full email sizes  
./run.sh compare-sizes

# Test response sizes with different limits
./run.sh size-test 20

# View detailed performance guide
./run.sh performance-guide
```

### Recommendations
- ✅ Use `fetch_email_headers` for listing/searching
- ✅ Use `fetch_email` only when you need content
- ⚠️ Avoid batch fetching full emails (memory intensive)
- 📊 Headers: 100 emails = ~43KB | Full emails: 100 emails = 1-8MB

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

## Limitations

- Single email account per server instance
- 25MB maximum attachment size
- IMAP search capabilities vary by provider
- No support for inline images in HTML emails (v1)

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

## License

MIT