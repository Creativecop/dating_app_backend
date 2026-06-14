package media

import (
	"time"

	"github.com/google/uuid"
)

const (
	MediaTypePhoto = "PHOTO"
	MediaTypeVideo = "VIDEO"

	PurposeProfilePhoto = "PROFILE_PHOTO"
	PurposeIntroVideo   = "INTRO_VIDEO"

	ProcessingUploaded   = "UPLOADED"
	ProcessingProcessing = "PROCESSING"
	ProcessingReady      = "READY"
	ProcessingFailed     = "FAILED"
	ProcessingDeleted    = "DELETED"

	ModerationPending  = "PENDING"
	ModerationApproved = "APPROVED"
	ModerationRejected = "REJECTED"
	ModerationFlagged  = "FLAGGED"

	VariantOriginal   = "ORIGINAL"
	VariantDisplay    = "DISPLAY"
	VariantThumbnail  = "THUMBNAIL"
	VariantBlurred    = "BLURRED"
	VariantTranscoded = "TRANSCODED"
)

type UserMedia struct {
	ID               uint64    `gorm:"primaryKey"`
	UUID             uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID           uint64
	MediaType        string
	MediaPurpose     string
	ProcessingStatus string
	ModerationStatus string
	IsPrimary        bool
	SortOrder        int

	OriginalFileName *string
	MimeType         *string
	SizeBytes        *int64
	Width            *int
	Height           *int
	DurationSeconds  *int
	ChecksumSHA256   *string

	ProcessingError *string
	RejectionReason *string

	UploadedAt  time.Time
	ProcessedAt *time.Time
	FailedAt    *time.Time
	ApprovedAt  *time.Time
	DeletedAt   *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	Variants  []UserMediaVariant `gorm:"foreignKey:MediaID"`
}

func (UserMedia) TableName() string {
	return "user_media"
}

type UserMediaVariant struct {
	ID              uint64    `gorm:"primaryKey"`
	UUID            uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	MediaID         uint64
	VariantType     string
	StorageProvider string
	Bucket          *string
	ObjectKey       string
	PublicURL       *string
	MimeType        *string
	SizeBytes       *int64
	Width           *int
	Height          *int
	DurationSeconds *int
	CreatedAt       time.Time
}

func (UserMediaVariant) TableName() string {
	return "user_media_variants"
}
