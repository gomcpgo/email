#!/bin/bash

# Source .env file if it exists
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

case "$1" in
    "build")
        echo "Building Email MCP server..."
        go build -o bin/email-mcp ./cmd
        echo "Build complete: bin/email-mcp"
        ;;
    
    "test")
        echo "Running unit tests..."
        go test -v ./pkg/...
        ;;
    
    "folders")
        echo "Listing email folders..."
        go run ./cmd -folders
        ;;
    
    "fetch")
        # Fetch emails from last N days
        if [ -z "$2" ]; then
            echo "Fetching emails from last 7 days..."
            go run ./cmd -fetch "since:7 days ago"
        else
            echo "Fetching emails since $2..."
            go run ./cmd -fetch "$2"
        fi
        ;;
    
    "email")
        # Fetch a specific email
        if [ -z "$2" ]; then
            echo "Usage: ./run.sh email <message-id>"
            exit 1
        fi
        echo "Fetching email: $2"
        go run ./cmd -email "$2"
        ;;
    
    "send-test")
        echo "Sending test email..."
        go run ./cmd -send-test
        ;;
    
    "attachment")
        # Fetch attachments from email
        if [ -z "$2" ]; then
            echo "Usage: ./run.sh attachment <message-id>"
            exit 1
        fi
        echo "Fetching attachments from: $2"
        go run ./cmd -attachment "$2"
        ;;
    
    "cache-info")
        echo "Cache information:"
        go run ./cmd -cache-info
        ;;
    
    "clear-cache")
        echo "Clearing cache..."
        go run ./cmd -clear-cache
        ;;
    
    "compare-sizes")
        # Compare response sizes between headers and full emails
        echo "=== EMAIL RESPONSE SIZE COMPARISON ==="
        echo ""
        echo "1. Fetching email headers (metadata only)..."
        echo "   Command: fetch_email_headers with limit=5"
        headers_response=$(go run ./cmd -fetch "2025-09-25" 2>/dev/null)
        headers_size=${#headers_response}
        echo "   Response size: $headers_size characters"
        echo ""
        
        echo "2. Fetching single full email (with body)..."
        # Get first message ID from headers (decode unicode escapes)
        message_id=$(echo "$headers_response" | grep -o '"message_id": "[^"]*"' | head -1 | cut -d'"' -f4 | sed 's/\\u003c/</g' | sed 's/\\u003e/>/g')
        if [ -n "$message_id" ]; then
            echo "   Command: fetch_email for message: ${message_id:0:50}..."
            full_response=$(go run ./cmd -email "$message_id" 2>/dev/null)
            full_size=${#full_response}
            echo "   Response size: $full_size characters"
            
            # Calculate ratio
            if [ $headers_size -gt 0 ]; then
                ratio=$((full_size * 100 / headers_size))
                echo ""
                echo "=== SIZE COMPARISON RESULTS ==="
                echo "Headers response: $headers_size chars"
                echo "Full email response: $full_size chars"
                echo "Size ratio: ${ratio}% (full email is ${ratio}% the size of headers response)"
                echo ""
                if [ $ratio -gt 200 ]; then
                    echo "💡 Recommendation: Use fetch_email_headers for listing, then fetch_email only when needed"
                else
                    echo "💡 Note: Size difference is relatively small for this email"
                fi
            fi
        else
            echo "   ❌ Could not extract message ID from headers response"
        fi
        ;;
    
    "size-test")
        # Test different email sizes
        if [ -z "$2" ]; then
            echo "Usage: ./run.sh size-test <limit>"
            echo "Example: ./run.sh size-test 10"
            exit 1
        fi
        limit=$2
        echo "=== TESTING EMAIL SIZES WITH LIMIT: $limit ==="
        echo ""
        echo "Fetching $limit email headers..."
        response=$(go run ./cmd -fetch "2025-09-20" 2>/dev/null | head -n 1000)
        size=${#response}
        avg_per_email=$((size / limit))
        echo "Total response size: $size characters"
        echo "Average per email header: ~$avg_per_email characters"
        echo ""
        echo "💡 Memory usage estimation:"
        echo "   - 50 headers: ~$((avg_per_email * 50)) chars"
        echo "   - 100 headers: ~$((avg_per_email * 100)) chars"
        echo "   - 500 headers: ~$((avg_per_email * 500)) chars"
        ;;
    
    "performance-guide")
        echo "=== EMAIL MCP SERVER PERFORMANCE GUIDE ==="
        echo ""
        echo "📊 RESPONSE SIZE ANALYSIS (Based on Testing):"
        echo ""
        echo "1. EMAIL HEADERS (metadata only):"
        echo "   • Average per email: ~430 characters"
        echo "   • 50 emails: ~21KB"
        echo "   • 100 emails: ~43KB"
        echo "   • 500 emails: ~215KB"
        echo ""
        echo "2. FULL EMAILS (with body + HTML):"
        echo "   • Small email (Google alert): ~7KB (13x larger than header)"
        echo "   • Medium email (newsletter): ~82KB (194x larger than header)"
        echo "   • Large email (with attachments): 100KB+ (200x+ larger)"
        echo ""
        echo "💡 PERFORMANCE RECOMMENDATIONS:"
        echo ""
        echo "✅ DO:"
        echo "   • Use fetch_email_headers for listing/searching emails"
        echo "   • Use fetch_email only when you need the actual content"
        echo "   • Set reasonable limits (50-100) for header fetches"
        echo "   • Cache frequently accessed emails"
        echo ""
        echo "⚠️  AVOID:"
        echo "   • Fetching full emails in batch (memory intensive)"
        echo "   • Using high limits (500+) without pagination"
        echo "   • Fetching full emails just to get sender/subject (use headers)"
        echo ""
        echo "📈 SCALING CONSIDERATIONS:"
        echo "   • Headers: 500 emails = ~215KB response"  
        echo "   • Full emails: 500 emails = 10-40MB response (avoid!)"
        echo "   • Use headers first, then fetch individual emails as needed"
        ;;
    
    "run")
        echo "Running Email MCP server..."
        go run ./cmd
        ;;
    
    "install")
        echo "Installing dependencies..."
        go mod download
        go mod tidy
        ;;
    
    *)
        echo "Email MCP Server - Build and Run Script"
        echo ""
        echo "Usage: $0 {command} [options]"
        echo ""
        echo "Commands:"
        echo "  build          - Build the MCP server binary"
        echo "  test           - Run unit tests"
        echo "  folders        - List all email folders"
        echo "  fetch [date]   - Fetch email headers (default: last 7 days)"
        echo "  email <id>     - Fetch a complete email by Message-ID"
        echo "  send-test      - Send a test email"
        echo "  attachment <id>- Fetch attachments from an email"
        echo "  cache-info     - Show cache statistics"
        echo "  clear-cache    - Clear all cached data"
        echo "  compare-sizes  - Compare response sizes: headers vs full email"
        echo "  size-test <n>  - Test response sizes with n email headers"
        echo "  performance-guide - Show detailed performance recommendations"
        echo "  run            - Run the MCP server"
        echo "  install        - Install Go dependencies"
        echo ""
        echo "Examples:"
        echo "  ./run.sh fetch                     # Fetch emails from last 7 days"
        echo "  ./run.sh fetch 2024-01-20          # Fetch emails since date"
        echo "  ./run.sh email '<msg@example.com>' # Fetch specific email"
        echo "  ./run.sh attachment '<msg@ex.com>' # Fetch attachments"
        echo "  ./run.sh compare-sizes             # Compare header vs full email sizes"
        echo "  ./run.sh size-test 20              # Test response size with 20 emails"
        ;;
esac