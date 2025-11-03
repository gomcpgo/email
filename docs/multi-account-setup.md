# Multi-Account Configuration Guide

This guide explains how to configure and use multiple email accounts with the Email MCP Server.

## Table of Contents

- [Overview](#overview)
- [Configuration](#configuration)
  - [Basic Setup](#basic-setup)
  - [Environment Variables](#environment-variables)
  - [Account Types](#account-types)
- [Using Accounts in Tool Calls](#using-accounts-in-tool-calls)
- [Storage Structure](#storage-structure)
- [Examples](#examples)

## Overview

The Email MCP Server supports managing multiple email accounts simultaneously. Each account:
- Has its own isolated storage (drafts, cache, attachments)
- Can use different email providers (Gmail, Outlook, custom IMAP/SMTP)
- Has independent IMAP/SMTP configurations
- Supports provider-specific auto-configuration

## Configuration

### Basic Setup

1. Copy the `.env.example` file to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Configure your accounts using the environment variable pattern
3. Set the default account ID
4. Start the server

### Environment Variables

All account-specific settings use the pattern: `ACCOUNT_{account_id}_{SETTING}`

#### Required Variables

For each account, you must set:

| Variable | Description | Example |
|----------|-------------|---------|
| `ACCOUNT_{id}_EMAIL` | Email address for the account | `ACCOUNT_work_EMAIL=user@company.com` |
| `ACCOUNT_{id}_PASSWORD` | App password or account password | `ACCOUNT_work_PASSWORD=xxxx` |
| `ACCOUNT_{id}_PROVIDER` | Provider type: `gmail`, `outlook`, or `custom` | `ACCOUNT_work_PROVIDER=gmail` |

#### Default Account

You must specify which account is the default:

```bash
DEFAULT_ACCOUNT_ID=work
```

This account is used when no `account_id` is specified in tool calls.

#### Optional Variables

For provider `gmail` and `outlook`, IMAP/SMTP settings are auto-configured. You can override them:

| Variable | Description | Default (Gmail) | Default (Outlook) |
|----------|-------------|-----------------|-------------------|
| `ACCOUNT_{id}_IMAP_SERVER` | IMAP server hostname | `imap.gmail.com` | `outlook.office365.com` |
| `ACCOUNT_{id}_IMAP_PORT` | IMAP server port | `993` | `993` |
| `ACCOUNT_{id}_SMTP_SERVER` | SMTP server hostname | `smtp.gmail.com` | `smtp-mail.outlook.com` |
| `ACCOUNT_{id}_SMTP_PORT` | SMTP server port | `587` | `587` |
| `ACCOUNT_{id}_TIMEOUT_SECONDS` | Operation timeout | `120` | `120` |

For `custom` provider, all IMAP/SMTP settings are **required**.

#### Global Settings

These settings apply to all accounts:

```bash
FILES_ROOT=/tmp/email-mcp              # Root directory for all account data
EMAIL_CACHE_MAX_SIZE=10485760          # 10MB cache limit per account
EMAIL_MAX_ATTACHMENT_SIZE=26214400     # 25MB max attachment size
```

### Account Types

#### Gmail Account

```bash
ACCOUNT_work_EMAIL=user@gmail.com
ACCOUNT_work_PASSWORD=your_app_password
ACCOUNT_work_PROVIDER=gmail
```

**Setup:**
1. Enable 2-factor authentication in Google Account
2. Generate app password: https://myaccount.google.com/apppasswords
3. Use the 16-character app password as `ACCOUNT_{id}_PASSWORD`

#### Outlook Account

```bash
ACCOUNT_personal_EMAIL=user@outlook.com
ACCOUNT_personal_PASSWORD=your_app_password
ACCOUNT_personal_PROVIDER=outlook
```

**Setup:**
1. Enable 2-factor authentication in Microsoft Account
2. Generate app password: https://account.microsoft.com/security
3. Use the app password as `ACCOUNT_{id}_PASSWORD`

#### Custom Email Server

```bash
ACCOUNT_custom_EMAIL=user@custom-domain.com
ACCOUNT_custom_PASSWORD=your_password
ACCOUNT_custom_PROVIDER=custom
ACCOUNT_custom_IMAP_SERVER=mail.custom-domain.com
ACCOUNT_custom_IMAP_PORT=993
ACCOUNT_custom_SMTP_SERVER=mail.custom-domain.com
ACCOUNT_custom_SMTP_PORT=587
```

For custom providers, you must specify all IMAP and SMTP settings.

## Using Accounts in Tool Calls

All MCP tools accept an optional `account_id` parameter to specify which account to use.

### Default Account Behavior

When `account_id` is **not specified**, the tool uses the account specified in `DEFAULT_ACCOUNT_ID`:

```json
{
  "folder": "INBOX",
  "limit": 10
}
```
☝️ Uses the default account (e.g., `work` if `DEFAULT_ACCOUNT_ID=work`)

### Explicit Account Selection

To use a specific account, include the `account_id` parameter:

```json
{
  "account_id": "personal",
  "folder": "INBOX",
  "limit": 10
}
```
☝️ Uses the `personal` account explicitly

### Tool Examples

#### List Folders

**Default account:**
```json
{
  "account_id": ""
}
```

**Specific account:**
```json
{
  "account_id": "work"
}
```

#### Fetch Email Headers

**Default account:**
```json
{
  "folder": "INBOX",
  "since_date": "2024-01-01",
  "limit": 50
}
```

**Work account:**
```json
{
  "account_id": "work",
  "folder": "INBOX",
  "since_date": "2024-01-01",
  "limit": 50
}
```

**Personal account:**
```json
{
  "account_id": "personal",
  "folder": "[Gmail]/All Mail",
  "unread_only": true,
  "limit": 20
}
```

#### Send Email

**Default account:**
```json
{
  "to": ["recipient@example.com"],
  "subject": "Hello",
  "body": "This is a test email"
}
```

**From specific account:**
```json
{
  "account_id": "work",
  "to": ["colleague@company.com"],
  "subject": "Work Update",
  "body": "Quarterly report attached",
  "attachments": ["cache_id_123"]
}
```

#### Create Draft

**Default account:**
```json
{
  "to": ["recipient@example.com"],
  "subject": "Draft Email",
  "body": "Draft content"
}
```

**Personal account:**
```json
{
  "account_id": "personal",
  "to": ["friend@example.com"],
  "subject": "Weekend Plans",
  "body": "Let's meet up this weekend"
}
```

#### Fetch Email

**Default account:**
```json
{
  "message_id": "<CADsK8=example@mail.gmail.com>"
}
```

**Specific account:**
```json
{
  "account_id": "work",
  "message_id": "<CADsK8=example@mail.gmail.com>"
}
```

## Storage Structure

Each account has its own isolated storage within the `FILES_ROOT` directory:

```
FILES_ROOT/
├── work/                           # Account: work
│   ├── drafts/
│   │   ├── draft_001.yaml
│   │   └── draft_002.yaml
│   ├── cache/
│   │   ├── emails/
│   │   │   ├── msg_abc123.yaml
│   │   │   └── msg_def456.yaml
│   │   └── attachments/
│   │       ├── att_hash1.pdf
│   │       └── att_hash2.jpg
│   └── metadata.yaml
│
├── personal/                       # Account: personal
│   ├── drafts/
│   ├── cache/
│   │   ├── emails/
│   │   └── attachments/
│   └── metadata.yaml
│
└── custom/                         # Account: custom
    ├── drafts/
    ├── cache/
    │   ├── emails/
    │   └── attachments/
    └── metadata.yaml
```

### Benefits of Isolated Storage

- **No data mixing**: Drafts and cached emails from different accounts never mix
- **Independent cache management**: Each account has its own cache size limit
- **Easy backup**: Back up individual accounts by copying their directories
- **Clear separation**: Easy to identify which account owns which data

## Examples

### Example 1: Basic Two-Account Setup

```bash
# .env file
DEFAULT_ACCOUNT_ID=work

# Work Gmail Account
ACCOUNT_work_EMAIL=john.doe@company.com
ACCOUNT_work_PASSWORD=abcd1234efgh5678
ACCOUNT_work_PROVIDER=gmail

# Personal Gmail Account
ACCOUNT_personal_EMAIL=johndoe@gmail.com
ACCOUNT_personal_PASSWORD=wxyz9876abcd5432
ACCOUNT_personal_PROVIDER=gmail

# Global Settings
FILES_ROOT=/home/user/.email-mcp
EMAIL_CACHE_MAX_SIZE=10485760
EMAIL_MAX_ATTACHMENT_SIZE=26214400
```

**Usage:**
```json
// Check work inbox (default account)
{
  "folder": "INBOX"
}

// Check personal inbox
{
  "account_id": "personal",
  "folder": "INBOX"
}
```

### Example 2: Three-Account Setup (Gmail + Outlook + Custom)

```bash
# .env file
DEFAULT_ACCOUNT_ID=work

# Work Gmail
ACCOUNT_work_EMAIL=employee@company.com
ACCOUNT_work_PASSWORD=gmail_app_password
ACCOUNT_work_PROVIDER=gmail

# Personal Outlook
ACCOUNT_personal_EMAIL=user@outlook.com
ACCOUNT_personal_PASSWORD=outlook_app_password
ACCOUNT_personal_PROVIDER=outlook

# Client Custom Server
ACCOUNT_client1_EMAIL=contact@client.com
ACCOUNT_client1_PASSWORD=client_password
ACCOUNT_client1_PROVIDER=custom
ACCOUNT_client1_IMAP_SERVER=mail.client.com
ACCOUNT_client1_IMAP_PORT=993
ACCOUNT_client1_SMTP_SERVER=mail.client.com
ACCOUNT_client1_SMTP_PORT=587

# Global Settings
FILES_ROOT=/var/lib/email-mcp
EMAIL_CACHE_MAX_SIZE=20971520
EMAIL_MAX_ATTACHMENT_SIZE=52428800
```

**Usage:**
```json
// Send from work account (default)
{
  "to": ["boss@company.com"],
  "subject": "Report",
  "body": "Please review"
}

// Send from personal account
{
  "account_id": "personal",
  "to": ["friend@example.com"],
  "subject": "Hello",
  "body": "How are you?"
}

// Send from client account
{
  "account_id": "client1",
  "to": ["contact@client.com"],
  "subject": "Project Update",
  "body": "Status update"
}
```

### Example 3: Switching Between Accounts

```json
// Fetch headers from work account
{
  "account_id": "work",
  "folder": "INBOX",
  "since_date": "2024-01-01"
}

// Fetch headers from personal account
{
  "account_id": "personal",
  "folder": "INBOX",
  "since_date": "2024-01-01"
}

// Send email from work account
{
  "account_id": "work",
  "to": ["colleague@company.com"],
  "subject": "Meeting",
  "body": "Let's meet tomorrow"
}

// Create draft in personal account
{
  "account_id": "personal",
  "to": ["friend@example.com"],
  "subject": "Weekend",
  "body": "Are you free this weekend?"
}
```

## Troubleshooting

### Account Not Found Error

**Error:** `account {id} not found`

**Solution:**
- Check that you've set `ACCOUNT_{id}_EMAIL` and `ACCOUNT_{id}_PASSWORD`
- Verify the account ID matches exactly (case-sensitive)
- Restart the server after adding new accounts

### Default Account Not Configured

**Error:** `default account {id} not found in configured accounts`

**Solution:**
- Ensure `DEFAULT_ACCOUNT_ID` matches an existing account ID
- Check that the default account has all required variables set

### Custom Provider Missing Settings

**Error:** `IMAP server not configured for account {id}`

**Solution:**
For `custom` providers, you must set:
- `ACCOUNT_{id}_IMAP_SERVER`
- `ACCOUNT_{id}_IMAP_PORT`
- `ACCOUNT_{id}_SMTP_SERVER`
- `ACCOUNT_{id}_SMTP_PORT`

### Authentication Failed

**Error:** `authentication failed`

**Solution:**
- For Gmail/Outlook: Use app passwords, not account passwords
- Verify 2-factor authentication is enabled
- Check that the app password is correct (no spaces)
- Ensure the account allows IMAP/SMTP access

## Best Practices

1. **Use Descriptive Account IDs**: Use clear names like `work`, `personal`, `client1` instead of `account1`, `account2`

2. **Set Appropriate Default**: Choose your most-used account as the default to minimize typing

3. **Secure Storage**: Keep your `.env` file secure and never commit it to version control

4. **Backup Regularly**: Back up your `FILES_ROOT` directory to prevent data loss

5. **Monitor Storage**: Each account has independent cache - monitor disk usage if managing many accounts

6. **Test Configuration**: Use terminal mode to test each account before using in production:
   ```bash
   ./run.sh folders    # Test default account
   ```

## Summary

- **Configure accounts**: Use `ACCOUNT_{id}_{SETTING}` pattern in `.env`
- **Set default**: Set `DEFAULT_ACCOUNT_ID` to your primary account
- **Use in tools**: Add `"account_id": "name"` to any tool call
- **Omit for default**: Leave out `account_id` to use the default account
- **Isolated storage**: Each account has separate drafts, cache, and attachments

For more information, see the main [README.md](../README.md).
