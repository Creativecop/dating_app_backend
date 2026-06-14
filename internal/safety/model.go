package safety

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	TargetUser    = "USER"
	TargetProfile = "PROFILE"
	TargetMessage = "MESSAGE"
	TargetMedia   = "MEDIA"
	TargetMatch   = "MATCH"

	ReasonFakeProfile    = "FAKE_PROFILE"
	ReasonHarassment     = "HARASSMENT"
	ReasonHateSpeech     = "HATE_SPEECH"
	ReasonSexualContent  = "NUDITY_OR_SEXUAL_CONTENT"
	ReasonScamSpam       = "SCAM_OR_SPAM"
	ReasonUnderage       = "UNDERAGE"
	ReasonViolenceThreat = "VIOLENCE_OR_THREAT"
	ReasonImpersonation  = "IMPERSONATION"
	ReasonOther          = "OTHER"

	ReportOpen     = "OPEN"
	ReportInReview = "IN_REVIEW"

	SeverityLow      = "LOW"
	SeverityMedium   = "MEDIUM"
	SeverityHigh     = "HIGH"
	SeverityCritical = "CRITICAL"

	CaseOpen     = "OPEN"
	CaseInReview = "IN_REVIEW"

	BlockSourceManual     = "MANUAL"
	BlockSourceReportFlow = "REPORT_FLOW"
	BlockSourceModeration = "MODERATION"

	ActionReportCreated = "REPORT_CREATED"
	ActionUserBlocked   = "USER_BLOCKED"
)

type ReportReason struct {
	ID          uint64    `gorm:"primaryKey"`
	UUID        uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	ReasonCode  string
	Title       string
	Description *string
	AppliesTo   []string `gorm:"type:text[]"`
	SortOrder   int
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (ReportReason) TableName() string {
	return "report_reasons"
}

type Report struct {
	ID               uint64    `gorm:"primaryKey"`
	UUID             uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	ReporterUserID   uint64
	ReportedUserID   *uint64
	TargetType       string
	TargetUUID       uuid.UUID
	ReasonID         uint64
	Note             *string
	EvidenceSnapshot datatypes.JSON
	Status           string
	Severity         string
	ModerationCaseID *uint64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (Report) TableName() string {
	return "reports"
}

type ModerationCase struct {
	ID              uint64    `gorm:"primaryKey"`
	UUID            uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	SubjectUserID   *uint64
	TargetType      *string
	TargetUUID      *uuid.UUID `gorm:"type:uuid"`
	Status          string
	Priority        string
	ReportCount     int
	OpenedAt        time.Time
	AssignedAdminID *uint64
	ResolvedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (ModerationCase) TableName() string {
	return "moderation_cases"
}

type ModerationAction struct {
	ID               uint64    `gorm:"primaryKey"`
	UUID             uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	ModerationCaseID *uint64
	ActorUserID      *uint64
	TargetUserID     *uint64
	ActionType       string
	Reason           *string
	Metadata         datatypes.JSON
	CreatedAt        time.Time
}

func (ModerationAction) TableName() string {
	return "moderation_actions"
}

type UserBlock struct {
	ID            uint64    `gorm:"primaryKey"`
	UUID          uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	BlockerUserID uint64
	BlockedUserID uint64
	Reason        *string
	ReasonCode    *string
	Note          *string
	Source        string
	ReportID      *uint64
	CreatedAt     time.Time
}

func (UserBlock) TableName() string {
	return "user_blocks"
}

type SafetySettings struct {
	ID                   uint64    `gorm:"primaryKey"`
	UUID                 uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID               uint64
	AllowMessageRequests bool
	AutoHideBlockedUsers bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (SafetySettings) TableName() string {
	return "safety_settings"
}
