# Email MCP Server

A Model Context Protocol (MCP) server for email operations via IMAP and SMTP. Supports Gmail, Outlook, and other IMAP/SMTP servers with multi-account support.

## Features

- **Multi-account support** - Manage multiple email accounts simultaneously
- **List email folders** - Enumerate all available folders/labels
- **Fetch email headers** - Get email metadata without downloading full content
- **Fetch complete emails** - Download full email with body and attachments
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

Example with account selection:
```json
{
  "account_id": "work",
  // ... other parameters
}
```

### list_folders
Lists all available email folders with message counts.

```json
{
  "account_id": "work"  // Optional: defaults to DEFAULT_ACCOUNT_ID
}
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