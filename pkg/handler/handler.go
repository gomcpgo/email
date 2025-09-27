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
	imapClient    *email.IMAPClient    // Lazy-initialized
	smtpClient    *email.SMTPClient    // Lazy-initialized
	attFetcher    *email.AttachmentFetcher // Lazy-initialized
	storage       *storage.Storage
	cacheManager  *storage.CacheManager
}

// NewHandler creates a new handler instance
func NewHandler(cfg *config.Config) (*Handler, error) {
	// Only create non-email clients at startup
	stor := storage.NewStorage(cfg.FilesRoot, cfg.CacheMaxSize)
	cacheManager := storage.NewCacheManager(cfg.FilesRoot, cfg.CacheMaxSize)

	return &Handler{
		config:       cfg,
		storage:      stor,
		cacheManager: cacheManager,
		// Email clients are lazy-initialized on first use
	}, nil
}

// getIMAPClient returns the IMAP client, initializing if necessary
func (h *Handler) getIMAPClient() (*email.IMAPClient, error) {
	if err := h.config.ValidateForOperation(); err != nil {
		return nil, err
	}
	
	if h.imapClient == nil {
		h.imapClient = email.NewIMAPClient(h.config)
	}
	return h.imapClient, nil
}

// getSMTPClient returns the SMTP client, initializing if necessary
func (h *Handler) getSMTPClient() (*email.SMTPClient, error) {
	if err := h.config.ValidateForOperation(); err != nil {
		return nil, err
	}
	
	if h.smtpClient == nil {
		h.smtpClient = email.NewSMTPClient(h.config)
	}
	return h.smtpClient, nil
}

// getAttachmentFetcher returns the attachment fetcher, initializing if necessary
func (h *Handler) getAttachmentFetcher() (*email.AttachmentFetcher, error) {
	imapClient, err := h.getIMAPClient()
	if err != nil {
		return nil, err
	}
	
	if h.attFetcher == nil {
		h.attFetcher = email.NewAttachmentFetcher(h.config, imapClient)
	}
	return h.attFetcher, nil
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
	case "create_draft":
		return h.handleCreateDraft(ctx, req.Arguments)
	case "list_drafts":
		return h.handleListDrafts(ctx, req.Arguments)
	case "get_draft":
		return h.handleGetDraft(ctx, req.Arguments)
	case "update_draft":
		return h.handleUpdateDraft(ctx, req.Arguments)
	case "send_draft":
		return h.handleSendDraft(ctx, req.Arguments)
	case "delete_draft":
		return h.handleDeleteDraft(ctx, req.Arguments)
	case "send_all_drafts":
		return h.handleSendAllDrafts(ctx, req.Arguments)
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