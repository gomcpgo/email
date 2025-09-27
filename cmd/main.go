package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gomcpgo/mcp/pkg/handler"
	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/gomcpgo/mcp/pkg/server"
	"github.com/prasanthmj/email/pkg/config"
	emailHandler "github.com/prasanthmj/email/pkg/handler"
	"github.com/prasanthmj/email/pkg/storage"
)

func main() {
	// Parse command line flags
	var (
		listFolders     = flag.Bool("folders", false, "List all email folders")
		fetchHeaders    = flag.String("fetch", "", "Fetch email headers: -fetch 'since:7 days ago'")
		fetchEmail      = flag.String("email", "", "Fetch full email by Message-ID")
		sendTest        = flag.Bool("send-test", false, "Send a test email")
		fetchAttachment = flag.String("attachment", "", "Fetch attachment: -attachment 'messageID'")
		cacheInfo       = flag.Bool("cache-info", false, "Show cache information")
		clearCache      = flag.Bool("clear-cache", false, "Clear all cache")
		debugMode       = flag.Bool("debug", false, "Enable debug mode")
		toolName        = flag.String("tool", "", "Call a specific tool")
		toolArgs        = flag.String("args", "{}", "Tool arguments as JSON")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Terminal mode operations
	if *listFolders || *fetchHeaders != "" || *fetchEmail != "" || *sendTest || 
	   *fetchAttachment != "" || *cacheInfo || *clearCache || *toolName != "" {
		err := runTerminalMode(cfg, *listFolders, *fetchHeaders, *fetchEmail, 
		                      *sendTest, *fetchAttachment, *cacheInfo, *clearCache, 
		                      *debugMode, *toolName, *toolArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// MCP Server mode (default)
	err = runMCPServer(cfg)
	if err != nil {
		log.Fatal(err)
	}
}

// runTerminalMode executes terminal mode for CLI testing
func runTerminalMode(cfg *config.Config, listFolders bool, fetchHeaders, fetchEmail string,
	sendTest bool, fetchAttachment string, cacheInfo, clearCache, debugMode bool, 
	toolName, toolArgs string) error {
	
	ctx := context.Background()
	
	// Create handler
	h, err := emailHandler.NewHandler(cfg)
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	// Cache operations
	if cacheInfo || clearCache {
		cacheManager := storage.NewCacheManager(cfg.FilesRoot, cfg.CacheMaxSize)
		
		if clearCache {
			if err := cacheManager.ClearCache(); err != nil {
				return fmt.Errorf("failed to clear cache: %w", err)
			}
			fmt.Println("Cache cleared successfully")
			return nil
		}
		
		if cacheInfo {
			info, err := cacheManager.GetCacheInfo()
			if err != nil {
				return fmt.Errorf("failed to get cache info: %w", err)
			}
			data, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(data))
			return nil
		}
	}

	// List folders
	if listFolders {
		req := &protocol.CallToolRequest{
			Name:      "list_folders",
			Arguments: map[string]interface{}{},
		}
		
		resp, err := h.CallTool(ctx, req)
		if err != nil {
			return err
		}
		
		if len(resp.Content) > 0 {
			fmt.Println(resp.Content[0].Text)
		}
		return nil
	}

	// Fetch email headers
	if fetchHeaders != "" {
		args := map[string]interface{}{}
		
		// Simple parsing for demo (format: "since:7 days ago")
		if fetchHeaders == "since:7 days ago" {
			args["since_date"] = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
		} else if fetchHeaders == "since:yesterday" {
			args["since_date"] = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		} else {
			// Assume it's a date
			args["since_date"] = fetchHeaders
		}
		
		req := &protocol.CallToolRequest{
			Name:      "fetch_email_headers",
			Arguments: args,
		}
		
		resp, err := h.CallTool(ctx, req)
		if err != nil {
			return err
		}
		
		if len(resp.Content) > 0 {
			fmt.Println(resp.Content[0].Text)
		}
		return nil
	}

	// Fetch full email
	if fetchEmail != "" {
		req := &protocol.CallToolRequest{
			Name: "fetch_email",
			Arguments: map[string]interface{}{
				"message_id": fetchEmail,
			},
		}
		
		resp, err := h.CallTool(ctx, req)
		if err != nil {
			return err
		}
		
		if len(resp.Content) > 0 {
			fmt.Println(resp.Content[0].Text)
		}
		return nil
	}

	// Send test email
	if sendTest {
		testAddr := os.Getenv("TEST_EMAIL_ADDRESS")
		if testAddr == "" {
			testAddr = cfg.EmailAddress // Send to self
		}
		
		req := &protocol.CallToolRequest{
			Name: "send_email",
			Arguments: map[string]interface{}{
				"to":      []string{testAddr},
				"subject": fmt.Sprintf("Test Email - %s", time.Now().Format("2006-01-02 15:04:05")),
				"body":    "This is a test email sent from the Email MCP server terminal mode.",
			},
		}
		
		resp, err := h.CallTool(ctx, req)
		if err != nil {
			return err
		}
		
		if len(resp.Content) > 0 {
			fmt.Println(resp.Content[0].Text)
		}
		return nil
	}

	// Fetch attachment
	if fetchAttachment != "" {
		req := &protocol.CallToolRequest{
			Name: "fetch_email_attachment",
			Arguments: map[string]interface{}{
				"message_id": fetchAttachment,
				"fetch_all":  true,
			},
		}
		
		resp, err := h.CallTool(ctx, req)
		if err != nil {
			return err
		}
		
		if len(resp.Content) > 0 {
			fmt.Println(resp.Content[0].Text)
		}
		return nil
	}

	// Generic tool invocation
	if toolName != "" {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(toolArgs), &args); err != nil {
			return fmt.Errorf("failed to parse tool arguments: %w", err)
		}
		
		req := &protocol.CallToolRequest{
			Name:      toolName,
			Arguments: args,
		}
		
		resp, err := h.CallTool(ctx, req)
		if err != nil {
			return err
		}
		
		if len(resp.Content) > 0 {
			fmt.Println(resp.Content[0].Text)
		}
		return nil
	}

	return nil
}

// runMCPServer runs the MCP server
func runMCPServer(cfg *config.Config) error {
	// Create handler
	h, err := emailHandler.NewHandler(cfg)
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	// Create handler registry
	registry := handler.NewHandlerRegistry()
	registry.RegisterToolHandler(h)

	// Create and run server
	srv := server.New(server.Options{
		Name:     "email-mcp-server",
		Version:  "1.0.0",
		Registry: registry,
	})

	fmt.Fprintf(os.Stderr, "Email MCP Server started\n")
	return srv.Run()
}