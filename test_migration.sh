#!/bin/bash

# Integration test for account folder migration
# This test simulates renaming an account from "Business" to "Operations"

set -e

TEST_DIR="/tmp/email-mcp-test"
ACCOUNT_EMAIL="test@example.com"
ACCOUNT_PASSWORD="test-password"

echo "=== Email MCP Migration Integration Test ==="
echo

# Clean up from previous test runs
echo "1. Cleaning up previous test data..."
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

# Set up initial account "Business"
echo "2. Setting up initial account 'Business'..."
export FILES_ROOT="$TEST_DIR"
export ACCOUNT_Business_EMAIL="$ACCOUNT_EMAIL"
export ACCOUNT_Business_PASSWORD="$ACCOUNT_PASSWORD"
export ACCOUNT_Business_PROVIDER="custom"
export ACCOUNT_Business_IMAP_SERVER="imap.example.com"
export ACCOUNT_Business_IMAP_PORT="993"
export ACCOUNT_Business_SMTP_SERVER="smtp.example.com"
export ACCOUNT_Business_SMTP_PORT="587"
export DEFAULT_ACCOUNT_ID="Business"

# Create some test data in Business folder
echo "3. Creating test data in Business account folder..."
mkdir -p "$TEST_DIR/Business/drafts"
echo "Draft email content" > "$TEST_DIR/Business/drafts/test-draft.txt"
mkdir -p "$TEST_DIR/Business/cache/emails"
echo "Cached email" > "$TEST_DIR/Business/cache/emails/test-cache.txt"

# Verify initial setup
if [ ! -d "$TEST_DIR/Business" ]; then
    echo "ERROR: Business folder was not created"
    exit 1
fi
echo "   ✓ Business folder exists"

# Load config (this will create metadata.yaml)
echo "4. Running MCP server to initialize metadata..."
timeout 2 ./bin/email-mcp 2>&1 | grep -q "Email MCP Server started" || true
sleep 1

# Check metadata was created
if [ ! -f "$TEST_DIR/Business/metadata.yaml" ]; then
    echo "ERROR: metadata.yaml was not created"
    exit 1
fi
echo "   ✓ Metadata file created"

# Verify metadata content
METADATA_CONTENT=$(cat "$TEST_DIR/Business/metadata.yaml")
if ! echo "$METADATA_CONTENT" | grep -q "account_id: Business"; then
    echo "ERROR: Metadata does not contain correct account_id"
    echo "$METADATA_CONTENT"
    exit 1
fi
if ! echo "$METADATA_CONTENT" | grep -q "email_address: $ACCOUNT_EMAIL"; then
    echo "ERROR: Metadata does not contain correct email_address"
    echo "$METADATA_CONTENT"
    exit 1
fi
echo "   ✓ Metadata content verified"

# Simulate renaming account from "Business" to "Operations"
echo "5. Renaming account from 'Business' to 'Operations'..."
unset ACCOUNT_Business_EMAIL
unset ACCOUNT_Business_PASSWORD
unset ACCOUNT_Business_PROVIDER
unset ACCOUNT_Business_IMAP_SERVER
unset ACCOUNT_Business_IMAP_PORT
unset ACCOUNT_Business_SMTP_SERVER
unset ACCOUNT_Business_SMTP_PORT

export ACCOUNT_Operations_EMAIL="$ACCOUNT_EMAIL"
export ACCOUNT_Operations_PASSWORD="$ACCOUNT_PASSWORD"
export ACCOUNT_Operations_PROVIDER="custom"
export ACCOUNT_Operations_IMAP_SERVER="imap.example.com"
export ACCOUNT_Operations_IMAP_PORT="993"
export ACCOUNT_Operations_SMTP_SERVER="smtp.example.com"
export ACCOUNT_Operations_SMTP_PORT="587"
export DEFAULT_ACCOUNT_ID="Operations"

# Restart MCP server (migration should happen)
echo "6. Restarting MCP server (migration should occur)..."
timeout 2 ./bin/email-mcp 2>&1 | tee /tmp/mcp-migration-output.log | grep -q "Email MCP Server started" || true
sleep 1

# Check migration output
if grep -q "Detected 1 account folder migration" /tmp/mcp-migration-output.log; then
    echo "   ✓ Migration detected"
else
    echo "ERROR: Migration was not detected"
    cat /tmp/mcp-migration-output.log
    exit 1
fi

if grep -q "All migrations completed successfully" /tmp/mcp-migration-output.log; then
    echo "   ✓ Migration completed successfully"
else
    echo "ERROR: Migration did not complete successfully"
    cat /tmp/mcp-migration-output.log
    exit 1
fi

# Verify Business folder is gone
if [ -d "$TEST_DIR/Business" ]; then
    echo "ERROR: Old 'Business' folder still exists after migration"
    exit 1
fi
echo "   ✓ Old 'Business' folder removed"

# Verify Operations folder exists
if [ ! -d "$TEST_DIR/Operations" ]; then
    echo "ERROR: New 'Operations' folder does not exist"
    exit 1
fi
echo "   ✓ New 'Operations' folder exists"

# Verify test data was preserved
if [ ! -f "$TEST_DIR/Operations/drafts/test-draft.txt" ]; then
    echo "ERROR: Draft file was not migrated"
    exit 1
fi
DRAFT_CONTENT=$(cat "$TEST_DIR/Operations/drafts/test-draft.txt")
if [ "$DRAFT_CONTENT" != "Draft email content" ]; then
    echo "ERROR: Draft file content was corrupted"
    exit 1
fi
echo "   ✓ Draft file migrated and preserved"

if [ ! -f "$TEST_DIR/Operations/cache/emails/test-cache.txt" ]; then
    echo "ERROR: Cache file was not migrated"
    exit 1
fi
CACHE_CONTENT=$(cat "$TEST_DIR/Operations/cache/emails/test-cache.txt")
if [ "$CACHE_CONTENT" != "Cached email" ]; then
    echo "ERROR: Cache file content was corrupted"
    exit 1
fi
echo "   ✓ Cache file migrated and preserved"

# Verify metadata was updated
if [ ! -f "$TEST_DIR/Operations/metadata.yaml" ]; then
    echo "ERROR: metadata.yaml was not migrated"
    exit 1
fi
NEW_METADATA=$(cat "$TEST_DIR/Operations/metadata.yaml")
if ! echo "$NEW_METADATA" | grep -q "account_id: Operations"; then
    echo "ERROR: Metadata account_id was not updated"
    echo "$NEW_METADATA"
    exit 1
fi
if ! echo "$NEW_METADATA" | grep -q "email_address: $ACCOUNT_EMAIL"; then
    echo "ERROR: Metadata email_address was changed (should remain same)"
    echo "$NEW_METADATA"
    exit 1
fi
echo "   ✓ Metadata updated correctly"

# Clean up
echo "7. Cleaning up test data..."
rm -rf "$TEST_DIR"
rm -f /tmp/mcp-migration-output.log

echo
echo "=== ✓ All migration tests passed! ==="
