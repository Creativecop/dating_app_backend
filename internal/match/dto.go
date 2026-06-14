package match

import "time"

type MatchListResponse struct {
	Items      []MatchListItem `json:"items"`
	NextCursor *string         `json:"nextCursor"`
}

type MatchListItem struct {
	MatchUUID string           `json:"matchUuid"`
	MatchedAt time.Time        `json:"matchedAt"`
	IsNew     bool             `json:"isNew"`
	User      MatchUserPreview `json:"user"`
}

type MatchUserPreview struct {
	UserUUID     string      `json:"userUuid"`
	ProfileUUID  *string     `json:"profileUuid"`
	DisplayName  *string     `json:"displayName"`
	Age          *int        `json:"age"`
	Bio          *string     `json:"bio"`
	PrimaryPhoto *MatchPhoto `json:"primaryPhoto"`
}

type MatchDetailResponse struct {
	MatchUUID string          `json:"matchUuid"`
	MatchedAt time.Time       `json:"matchedAt"`
	IsNew     bool            `json:"isNew"`
	User      MatchUserDetail `json:"user"`
}

type MatchUserDetail struct {
	UserUUID         string                 `json:"userUuid"`
	ProfileUUID      *string                `json:"profileUuid"`
	DisplayName      *string                `json:"displayName"`
	Age              *int                   `json:"age"`
	Bio              *string                `json:"bio"`
	Photos           []MatchPhoto           `json:"photos"`
	Interests        []string               `json:"interests"`
	Prompts          []MatchPromptAnswer    `json:"prompts"`
	LifestyleAnswers []MatchLifestyleAnswer `json:"lifestyleAnswers"`
}

type MatchPhoto struct {
	MediaUUID    string `json:"mediaUuid"`
	DisplayURL   string `json:"displayUrl,omitempty"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
}

type MatchPromptAnswer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type MatchLifestyleAnswer struct {
	QuestionKey string `json:"questionKey"`
	Question    string `json:"question"`
	Answer      any    `json:"answer"`
}

type MarkSeenResponse struct {
	MatchUUID    string     `json:"matchUuid"`
	SeenAt       *time.Time `json:"seenAt"`
	LastOpenedAt time.Time  `json:"lastOpenedAt"`
}

type UnmatchRequest struct {
	ReasonCode *string `json:"reasonCode"`
	ReasonNote *string `json:"reasonNote"`
}

type UnmatchResponse struct {
	MatchUUID string `json:"matchUuid"`
	Status    string `json:"status"`
}
