package email

import "time"

// EmailHeader represents email metadata without body
type EmailHeader struct {
	MessageID      string    `yaml:"message_id" json:"message_id"`
	Folder         string    `yaml:"folder" json:"folder"`
	From           string    `yaml:"from" json:"from"`
	To             []string  `yaml:"to" json:"to"`
	CC             []string  `yaml:"cc,omitempty" json:"cc,omitempty"`
	Subject        string    `yaml:"subject" json:"subject"`
	Date           time.Time `yaml:"date" json:"date"`
	HasAttachments bool      `yaml:"has_attachments" json:"has_attachments"`
	IsUnread       bool      `yaml:"is_unread" json:"is_unread"`
	Size           int64     `yaml:"size,omitempty" json:"size,omitempty"`
}

// Email represents a full email with body
type Email struct {
	MessageID      string       `yaml:"message_id" json:"message_id"`
	Folder         string       `yaml:"folder" json:"folder"`
	From           string       `yaml:"from" json:"from"`
	To             []string     `yaml:"to" json:"to"`
	CC             []string     `yaml:"cc,omitempty" json:"cc,omitempty"`
	BCC            []string     `yaml:"bcc,omitempty" json:"bcc,omitempty"`
	Subject        string       `yaml:"subject" json:"subject"`
	Date           time.Time    `yaml:"date" json:"date"`
	Body           string       `yaml:"body" json:"body"`
	HTMLBody       string       `yaml:"html_body,omitempty" json:"html_body,omitempty"`
	Attachments    []Attachment `yaml:"attachments,omitempty" json:"attachments,omitempty"`
	InReplyTo      string       `yaml:"in_reply_to,omitempty" json:"in_reply_to,omitempty"`
	References     []string     `yaml:"references,omitempty" json:"references,omitempty"`
	CachedAt       time.Time    `yaml:"cached_at,omitempty" json:"-"`
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string `yaml:"filename" json:"filename"`
	Size        int64  `yaml:"size" json:"size"`
	ContentType string `yaml:"content_type,omitempty" json:"content_type,omitempty"`
	CacheID     string `yaml:"cache_id,omitempty" json:"cache_id,omitempty"`
}

// FetchOptions represents email fetching parameters
type FetchOptions struct {
	Folder          string    `json:"folder"`
	SinceDate       time.Time `json:"since_date"`
	UntilDate       time.Time `json:"until_date"`
	From            string    `json:"from"`
	SubjectContains string    `json:"subject_contains"`
	UnreadOnly      bool      `json:"unread_only"`
	Limit           int       `json:"limit"`
}

// SendOptions represents email sending parameters
type SendOptions struct {
	To               []string `json:"to"`
	CC               []string `json:"cc"`
	BCC              []string `json:"bcc"`
	Subject          string   `json:"subject"`
	Body             string   `json:"body"`
	HTMLBody         string   `json:"html_body"`
	Attachments      []string `json:"attachments"` // Cache IDs
	ReplyToMessageID string   `json:"reply_to_message_id"`
	References       []string `json:"references"`
}

// Folder represents an IMAP folder
type Folder struct {
	Name         string `json:"name"`
	MessageCount uint32 `json:"message_count"`
	UnreadCount  uint32 `json:"unread_count"`
}