package handler

import (
	"context"
	"fmt"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/prasanthmj/email/pkg/config"
	"github.com/prasanthmj/email/pkg/email"
	"github.com/prasanthmj/email/pkg/storage"
)

// AccountClients holds per-account client instances
type AccountClients struct {
	imapClient   *email.IMAPClient
	smtpClient   *email.SMTPClient
	attFetcher   *email.AttachmentFetcher
	storage      *storage.Storage
	cacheManager *storage.CacheManager
}

// Handler handles MCP protocol operations
type Handler struct {
	config  *config.MultiAccountConfig
	clients map[string]*AccountClients // Per-account clients (lazy-initialized)
}

// NewHandler creates a new handler instance
func NewHandler(cfg *config.MultiAccountConfig) (*Handler, error) {
	return &Handler{
		config:  cfg,
		clients: make(map[string]*AccountClients),
	}, nil
}

// resolveAccountID returns the actual account ID to use (default if empty)
func (h *Handler) resolveAccountID(requestedID string) string {
	if requestedID == "" {
		return h.config.DefaultAccountID
	}
	return requestedID
}

// getAccountClients returns or creates the account clients for the given account ID
func (h *Handler) getAccountClients(accountID string) (*AccountClients, *config.AccountConfig, error) {
	accountID = h.resolveAccountID(accountID)

	// Get account config
	acctCfg, err := h.config.GetAccount(accountID)
	if err != nil {
		return nil, nil, err
	}

	// Check if clients already exist
	if clients, ok := h.clients[accountID]; ok {
		return clients, acctCfg, nil
	}

	// Create new clients for this account
	clients := &AccountClients{
		storage:      storage.NewStorage(acctCfg.DraftsDir[:len(acctCfg.DraftsDir)-len("/drafts")], h.config.CacheMaxSize),
		cacheManager: storage.NewCacheManager(acctCfg.DraftsDir[:len(acctCfg.DraftsDir)-len("/drafts")], h.config.CacheMaxSize),
	}

	h.clients[accountID] = clients
	return clients, acctCfg, nil
}

// getIMAPClient returns the IMAP client for the account, initializing if necessary
func (h *Handler) getIMAPClient(accountID string) (*email.IMAPClient, error) {
	clients, acctCfg, err := h.getAccountClients(accountID)
	if err != nil {
		return nil, err
	}

	if err := acctCfg.ValidateForOperation(); err != nil {
		return nil, err
	}

	if clients.imapClient == nil {
		clients.imapClient = email.NewIMAPClient(acctCfg)
	}
	return clients.imapClient, nil
}

// getSMTPClient returns the SMTP client for the account, initializing if necessary
func (h *Handler) getSMTPClient(accountID string) (*email.SMTPClient, error) {
	clients, acctCfg, err := h.getAccountClients(accountID)
	if err != nil {
		return nil, err
	}

	if err := acctCfg.ValidateForOperation(); err != nil {
		return nil, err
	}

	if clients.smtpClient == nil {
		clients.smtpClient = email.NewSMTPClient(acctCfg)
	}
	return clients.smtpClient, nil
}

// getAttachmentFetcher returns the attachment fetcher for the account, initializing if necessary
func (h *Handler) getAttachmentFetcher(accountID string) (*email.AttachmentFetcher, error) {
	clients, acctCfg, err := h.getAccountClients(accountID)
	if err != nil {
		return nil, err
	}

	imapClient, err := h.getIMAPClient(accountID)
	if err != nil {
		return nil, err
	}

	if clients.attFetcher == nil {
		clients.attFetcher = email.NewAttachmentFetcher(acctCfg, imapClient)
	}
	return clients.attFetcher, nil
}

// getStorage returns the storage for the account
func (h *Handler) getStorage(accountID string) (*storage.Storage, error) {
	clients, _, err := h.getAccountClients(accountID)
	if err != nil {
		return nil, err
	}
	return clients.storage, nil
}

// getCacheManager returns the cache manager for the account
func (h *Handler) getCacheManager(accountID string) (*storage.CacheManager, error) {
	clients, _, err := h.getAccountClients(accountID)
	if err != nil {
		return nil, err
	}
	return clients.cacheManager, nil
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