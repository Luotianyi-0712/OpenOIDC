package domain

import (
	"time"

	"github.com/google/uuid"
)

// RiskReport represents a developer's report of an abusive user.
type RiskReport struct {
	ID          uuid.UUID  `json:"id"`
	ClientID    uuid.UUID  `json:"client_id"`    // Which app reported
	ReporterID  uuid.UUID  `json:"reporter_id"`  // Developer who reported
	TargetID    uuid.UUID  `json:"target_id"`    // User being reported
	Reason      string     `json:"reason"`       // Freeform reason
	Category    string     `json:"category"`     // spam, abuse, fraud, bot, other
	Status      string     `json:"status"`       // pending, confirmed, dismissed
	AdminNote   string     `json:"admin_note"`   // Admin's comment
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy  *uuid.UUID `json:"resolved_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

const (
	ReportStatusPending   = "pending"
	ReportStatusConfirmed = "confirmed"
	ReportStatusDismissed = "dismissed"

	ReportCategorySpam  = "spam"
	ReportCategoryAbuse = "abuse"
	ReportCategoryFraud = "fraud"
	ReportCategoryBot   = "bot"
	ReportCategoryOther = "other"
)

// RiskListEntry represents a blacklisted social account.
// When a provider+provider_uid is in this list, new registrations/bindings with it are blocked.
type RiskListEntry struct {
	ID          uuid.UUID  `json:"id"`
	Provider    string     `json:"provider"`
	ProviderUID string     `json:"provider_uid"`
	UserID      *uuid.UUID `json:"user_id,omitempty"` // The user who had this binding when flagged
	Reason      string     `json:"reason"`
	ReportID    *uuid.UUID `json:"report_id,omitempty"` // Linked report
	AddedBy     *uuid.UUID `json:"added_by,omitempty"`  // Admin or system
	CreatedAt   time.Time  `json:"created_at"`
}

// UserAuthorization represents a user's active authorization to a third-party app.
type UserAuthorization struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	ClientID    string    `json:"client_id"`
	ClientName  string    `json:"client_name"`
	Scopes      []string  `json:"scopes"`
	GrantedAt   time.Time `json:"granted_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
}
