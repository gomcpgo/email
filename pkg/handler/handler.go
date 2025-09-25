package handler

import (
	"context"
	"fmt"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/prasanthmj/email/pkg/config"
	"github.com/prasanthmj/email/pkg/email"
	"github.com/prasanthmj/email/pkg/storage"
)

// Handler handles MCP protocol operations
type Handler struct {
	config        *config.Config
	imapClient    *email.IMAPClient
	smtpClient    *email.SMTPClient
	attFetcher    *email.AttachmentFetcher
	storage       *storage.Storage
	cacheManager  *storage.CacheManager
}

// NewHandler creates a new handler instance
func NewHandler(cfg *config.Config) (*Handler, error) {
	imapClient := email.NewIMAPClient(cfg)
	smtpClient := email.NewSMTPClient(cfg)
	attFetcher := email.NewAttachmentFetcher(cfg, imapClient)
	stor := storage.NewStorage(cfg.FilesRoot, cfg.CacheMaxSize)
	cacheManager := storage.NewCacheManager(cfg.FilesRoot, cfg.CacheMaxSize)

	return &Handler{
		config:       cfg,
		imapClient:   imapClient,
		smtpClient:   smtpClient,
		attFetcher:   attFetcher,
		storage:      stor,
		cacheManager: cacheManager,
	}, nil
}

// CallTool handles MCP tool calls
func (h *Handler) CallTool(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResponse, error) {
	switch req.Name {
	case "list_folders":
		return h.handleListFolders(ctx, req.Arguments)
	case "fetch_email_headers":
		return h.handleFetchEmailHeaders(ctx, req.Arguments)
	case "fetch_email":
		return h.handleFetchEmail(ctx, req.Arguments)
	case "send_email":
		return h.handleSendEmail(ctx, req.Arguments)
	case "fetch_email_attachment":
		return h.handleFetchEmailAttachment(ctx, req.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", req.Name)
	}
}

// ListTools returns available tools
func (h *Handler) ListTools(ctx context.Context) (*protocol.ListToolsResponse, error) {
	return &protocol.ListToolsResponse{
		Tools: GetTools(),
	}, nil
}