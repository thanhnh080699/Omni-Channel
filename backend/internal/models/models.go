package models

import "time"

type Base struct {
	ID        string     `bson:"_id" json:"id"`
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

type User struct {
	Base         `bson:",inline"`
	Email        string     `bson:"email" json:"email"`
	PasswordHash string     `bson:"password_hash" json:"-"`
	DisplayName  string     `bson:"display_name" json:"display_name"`
	Status       string     `bson:"status" json:"status"`
	RoleIDs      []string   `bson:"role_ids" json:"role_ids"`
	TeamIDs      []string   `bson:"team_ids" json:"team_ids"`
	LastLoginAt  *time.Time `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`
}

type Role struct {
	Base            `bson:",inline"`
	Name            string   `bson:"name" json:"name"`
	Code            string   `bson:"code" json:"code"`
	PermissionCodes []string `bson:"permission_codes" json:"permission_codes"`
	IsSystem        bool     `bson:"is_system" json:"is_system"`
}

type Permission struct {
	Base        `bson:",inline"`
	Code        string `bson:"code" json:"code"`
	Resource    string `bson:"resource" json:"resource"`
	Action      string `bson:"action" json:"action"`
	Description string `bson:"description" json:"description"`
}

type Team struct {
	Base           `bson:",inline"`
	Name           string   `bson:"name" json:"name"`
	ParentTeamID   string   `bson:"parent_team_id,omitempty" json:"parent_team_id,omitempty"`
	ManagerUserIDs []string `bson:"manager_user_ids" json:"manager_user_ids"`
	Status         string   `bson:"status" json:"status"`
}

type TeamMember struct {
	Base          `bson:",inline"`
	TeamID        string    `bson:"team_id" json:"team_id"`
	UserID        string    `bson:"user_id" json:"user_id"`
	MemberRole    string    `bson:"member_role" json:"member_role"`
	ManagerUserID string    `bson:"manager_user_id,omitempty" json:"manager_user_id,omitempty"`
	JoinedAt      time.Time `bson:"joined_at" json:"joined_at"`
}

type Channel struct {
	Base                 `bson:",inline"`
	Code                 string `bson:"code" json:"code"`
	Name                 string `bson:"name" json:"name"`
	Kind                 string `bson:"kind" json:"kind"`
	OfficialAPIAvailable bool   `bson:"official_api_available" json:"official_api_available"`
	Status               string `bson:"status" json:"status"`
}

type ChannelAccount struct {
	Base             `bson:",inline"`
	ChannelID        string     `bson:"channel_id" json:"channel_id"`
	Name             string     `bson:"name" json:"name"`
	OwnerTeamID      string     `bson:"owner_team_id,omitempty" json:"owner_team_id,omitempty"`
	CredentialRef    string     `bson:"credential_ref,omitempty" json:"credential_ref,omitempty"`
	WebhookSecretRef string     `bson:"webhook_secret_ref,omitempty" json:"webhook_secret_ref,omitempty"`
	SessionStatus    string     `bson:"session_status" json:"session_status"`
	LastWebhookAt    *time.Time `bson:"last_webhook_at,omitempty" json:"last_webhook_at,omitempty"`
	LastSyncAt       *time.Time `bson:"last_sync_at,omitempty" json:"last_sync_at,omitempty"`
	LastError        string     `bson:"last_error,omitempty" json:"last_error,omitempty"`
	Enabled          bool       `bson:"enabled" json:"enabled"`
}

type Conversation struct {
	Base                   `bson:",inline"`
	ChannelAccountID       string    `bson:"channel_account_id" json:"channel_account_id"`
	ExternalConversationID string    `bson:"external_conversation_id" json:"external_conversation_id"`
	CustomerRef            string    `bson:"customer_ref,omitempty" json:"customer_ref,omitempty"`
	AssignedUserID         string    `bson:"assigned_user_id,omitempty" json:"assigned_user_id,omitempty"`
	AssignedTeamID         string    `bson:"assigned_team_id,omitempty" json:"assigned_team_id,omitempty"`
	Status                 string    `bson:"status" json:"status"`
	LastMessageAt          time.Time `bson:"last_message_at" json:"last_message_at"`
	UnreadCount            int       `bson:"unread_count" json:"unread_count"`
	Tags                   []string  `bson:"tags" json:"tags"`
}

type ConversationMember struct {
	Base              `bson:",inline"`
	ConversationID    string `bson:"conversation_id" json:"conversation_id"`
	UserID            string `bson:"user_id" json:"user_id"`
	AccessLevel       string `bson:"access_level" json:"access_level"`
	Source            string `bson:"source" json:"source"`
	LastSeenMessageID string `bson:"last_seen_message_id,omitempty" json:"last_seen_message_id,omitempty"`
}

type Message struct {
	Base              `bson:",inline"`
	ConversationID    string     `bson:"conversation_id" json:"conversation_id"`
	Direction         string     `bson:"direction" json:"direction"`
	SenderType        string     `bson:"sender_type" json:"sender_type"`
	SenderUserID      string     `bson:"sender_user_id,omitempty" json:"sender_user_id,omitempty"`
	ChannelMessageID  string     `bson:"channel_message_id,omitempty" json:"channel_message_id,omitempty"`
	ChannelMessageKey string     `bson:"channel_message_key,omitempty" json:"channel_message_key,omitempty"`
	Text              string     `bson:"text" json:"text"`
	Status            string     `bson:"status" json:"status"`
	EventTime         time.Time  `bson:"event_time" json:"event_time"`
	SentAt            *time.Time `bson:"sent_at,omitempty" json:"sent_at,omitempty"`
	DeliveredAt       *time.Time `bson:"delivered_at,omitempty" json:"delivered_at,omitempty"`
	ReadAt            *time.Time `bson:"read_at,omitempty" json:"read_at,omitempty"`
}

type MessageAttachment struct {
	Base           `bson:",inline"`
	MessageID      string `bson:"message_id" json:"message_id"`
	ConversationID string `bson:"conversation_id" json:"conversation_id"`
	Type           string `bson:"type" json:"type"`
	OriginalURL    string `bson:"original_url,omitempty" json:"original_url,omitempty"`
	CDNPath        string `bson:"cdn_path,omitempty" json:"cdn_path,omitempty"`
	CDNURL         string `bson:"cdn_url,omitempty" json:"cdn_url,omitempty"`
	Status         string `bson:"status" json:"status"`
	SizeBytes      int64  `bson:"size_bytes,omitempty" json:"size_bytes,omitempty"`
	MimeType       string `bson:"mime_type,omitempty" json:"mime_type,omitempty"`
	Error          string `bson:"error,omitempty" json:"error,omitempty"`
}

type InboundEvent struct {
	Base              `bson:",inline"`
	ChannelAccountID  string                 `bson:"channel_account_id" json:"channel_account_id"`
	EventID           string                 `bson:"event_id" json:"event_id"`
	IdempotencyKey    string                 `bson:"idempotency_key" json:"idempotency_key"`
	RawPayload        map[string]interface{} `bson:"raw_payload" json:"raw_payload"`
	Status            string                 `bson:"status" json:"status"`
	EventTime         time.Time              `bson:"event_time" json:"event_time"`
	GatewayReceivedAt time.Time              `bson:"gateway_received_at" json:"gateway_received_at"`
	QueuedAt          *time.Time             `bson:"queued_at,omitempty" json:"queued_at,omitempty"`
	ProcessedAt       *time.Time             `bson:"processed_at,omitempty" json:"processed_at,omitempty"`
	AttemptCount      int                    `bson:"attempt_count" json:"attempt_count"`
	Error             string                 `bson:"error,omitempty" json:"error,omitempty"`
}

type OutboundEvent struct {
	Base             `bson:",inline"`
	MessageID        string     `bson:"message_id" json:"message_id"`
	ChannelAccountID string     `bson:"channel_account_id" json:"channel_account_id"`
	IdempotencyKey   string     `bson:"idempotency_key" json:"idempotency_key"`
	Status           string     `bson:"status" json:"status"`
	AttemptCount     int        `bson:"attempt_count" json:"attempt_count"`
	LastError        string     `bson:"last_error,omitempty" json:"last_error,omitempty"`
	SentAt           *time.Time `bson:"sent_at,omitempty" json:"sent_at,omitempty"`
}

type QueueJob struct {
	Base         `bson:",inline"`
	QueueName    string     `bson:"queue_name" json:"queue_name"`
	JobType      string     `bson:"job_type" json:"job_type"`
	RefID        string     `bson:"ref_id" json:"ref_id"`
	Status       string     `bson:"status" json:"status"`
	AttemptCount int        `bson:"attempt_count" json:"attempt_count"`
	NextRunAt    *time.Time `bson:"next_run_at,omitempty" json:"next_run_at,omitempty"`
	LastError    string     `bson:"last_error,omitempty" json:"last_error,omitempty"`
}

type SyncCheckpoint struct {
	Base             `bson:",inline"`
	ChannelAccountID string     `bson:"channel_account_id" json:"channel_account_id"`
	ConversationID   string     `bson:"conversation_id,omitempty" json:"conversation_id,omitempty"`
	Cursor           string     `bson:"cursor,omitempty" json:"cursor,omitempty"`
	LastSyncedAt     *time.Time `bson:"last_synced_at,omitempty" json:"last_synced_at,omitempty"`
	Status           string     `bson:"status" json:"status"`
	LastError        string     `bson:"last_error,omitempty" json:"last_error,omitempty"`
}

type AuditLog struct {
	Base         `bson:",inline"`
	ActorUserID  string                 `bson:"actor_user_id" json:"actor_user_id"`
	Action       string                 `bson:"action" json:"action"`
	ResourceType string                 `bson:"resource_type" json:"resource_type"`
	ResourceID   string                 `bson:"resource_id" json:"resource_id"`
	Metadata     map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	IP           string                 `bson:"ip,omitempty" json:"ip,omitempty"`
	UserAgent    string                 `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
}
