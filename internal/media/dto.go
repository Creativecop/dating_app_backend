package media

type MediaResponse struct {
	MediaUUID        string                 `json:"mediaUuid"`
	MediaType        string                 `json:"mediaType"`
	MediaPurpose     string                 `json:"mediaPurpose"`
	ProcessingStatus string                 `json:"processingStatus"`
	ModerationStatus string                 `json:"moderationStatus"`
	IsPrimary        bool                   `json:"isPrimary"`
	SortOrder        int                    `json:"sortOrder"`
	OriginalFileName *string                `json:"originalFileName,omitempty"`
	MimeType         *string                `json:"mimeType,omitempty"`
	SizeBytes        *int64                 `json:"sizeBytes,omitempty"`
	Width            *int                   `json:"width,omitempty"`
	Height           *int                   `json:"height,omitempty"`
	DurationSeconds  *int                   `json:"durationSeconds,omitempty"`
	ProcessingError  *string                `json:"processingError,omitempty"`
	RejectionReason  *string                `json:"rejectionReason,omitempty"`
	Variants         []MediaVariantResponse `json:"variants"`
}

type MediaVariantResponse struct {
	Type     string  `json:"type"`
	URL      string  `json:"url"`
	MimeType *string `json:"mimeType,omitempty"`
	Width    *int    `json:"width,omitempty"`
	Height   *int    `json:"height,omitempty"`
}

type MediaListResponse struct {
	Items []MediaResponse `json:"items"`
}

type ReorderRequest struct {
	Items []ReorderItemRequest `json:"items" binding:"required"`
}

type ReorderItemRequest struct {
	MediaUUID string `json:"mediaUuid" binding:"required"`
	SortOrder int    `json:"sortOrder" binding:"required"`
}

type UploadResult struct {
	Response MediaResponse
	Accepted bool
}

type ServeResult struct {
	MimeType  string
	ObjectKey string
	LocalPath string
	AccelPath string
}
