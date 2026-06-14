package discovery

import (
	"encoding/json"
	"time"
)

type UpdatePreferencesRequest struct {
	MinAge            *int     `json:"minAge"`
	MaxAge            *int     `json:"maxAge"`
	PreferredGenders  []string `json:"preferredGenders"`
	MaxDistanceKM     *int     `json:"maxDistanceKm"`
	VerifiedOnly      *bool    `json:"verifiedOnly"`
	ShowMeInDiscovery *bool    `json:"showMeInDiscovery"`
	HideDistance      *bool    `json:"hideDistance"`

	present map[string]bool
	nulls   map[string]bool
}

func (r *UpdatePreferencesRequest) UnmarshalJSON(data []byte) error {
	type alias UpdatePreferencesRequest
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*r = UpdatePreferencesRequest(aux)
	r.present = make(map[string]bool, len(raw))
	r.nulls = make(map[string]bool)
	for key, value := range raw {
		r.present[key] = true
		if string(value) == "null" {
			r.nulls[key] = true
		}
	}
	return nil
}

func (r UpdatePreferencesRequest) Has(field string) bool {
	return r.present[field]
}

func (r UpdatePreferencesRequest) IsNull(field string) bool {
	return r.nulls[field]
}

type PreferencesResponse struct {
	UUID              string     `json:"uuid"`
	MinAge            int        `json:"minAge"`
	MaxAge            int        `json:"maxAge"`
	PreferredGenders  []string   `json:"preferredGenders"`
	MaxDistanceKM     int        `json:"maxDistanceKm"`
	VerifiedOnly      bool       `json:"verifiedOnly"`
	ShowMeInDiscovery bool       `json:"showMeInDiscovery"`
	HideDistance      bool       `json:"hideDistance"`
	IsDefault         bool       `json:"isDefault"`
	CustomizedAt      *time.Time `json:"customizedAt"`
	ActivatedAt       *time.Time `json:"activatedAt"`
}

type ReadinessResponse struct {
	DiscoveryEligible bool     `json:"discoveryEligible"`
	Missing           []string `json:"missing"`
	Blocked           []string `json:"blocked"`
}

type FeedResponse struct {
	Items      []DiscoveryProfilePreview `json:"items"`
	NextCursor *string                   `json:"nextCursor"`
}

type DiscoveryProfilePreview struct {
	UserUUID       string                  `json:"userUuid"`
	ProfileUUID    string                  `json:"profileUuid"`
	DisplayName    *string                 `json:"displayName"`
	Age            *int                    `json:"age"`
	DistanceKM     *int                    `json:"distanceKm"`
	DistanceHidden bool                    `json:"distanceHidden"`
	Bio            *string                 `json:"bio"`
	PrimaryPhoto   DiscoveryPrimaryPhoto   `json:"primaryPhoto"`
	Interests      []string                `json:"interests"`
	Prompts        []DiscoveryPromptAnswer `json:"prompts"`
}

type DiscoveryPrimaryPhoto struct {
	MediaUUID    string `json:"mediaUuid"`
	DisplayURL   string `json:"displayUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

type DiscoveryPromptAnswer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type DiscoverableTargetSnapshot struct {
	TargetUserID      uint64
	TargetUserUUID    string
	TargetProfileUUID string
	DistanceMeters    int
}

type BlockUserRequest struct {
	Reason *string `json:"reason"`
}

type BlockedUserResponse struct {
	UserUUID    string    `json:"userUuid"`
	DisplayName *string   `json:"displayName"`
	BlockedAt   time.Time `json:"blockedAt"`
}

type BlockListResponse struct {
	Items []BlockedUserResponse `json:"items"`
}

type CreateImpressionsRequest struct {
	Source string                    `json:"source"`
	Items  []CreateImpressionRequest `json:"items" binding:"required"`
}

type CreateImpressionRequest struct {
	CandidateUserUUID string `json:"candidateUserUuid" binding:"required"`
	RankPosition      int    `json:"rankPosition" binding:"required"`
}

type CreateImpressionsResponse struct {
	Inserted int `json:"inserted"`
}

type CreateActionRequest struct {
	TargetUserUUID string `json:"targetUserUuid" binding:"required"`
	ActionType     string `json:"actionType" binding:"required"`
	ClientActionID string `json:"clientActionId" binding:"required"`
}

type ActionResponse struct {
	ActionType string         `json:"actionType"`
	Matched    bool           `json:"matched"`
	Match      *MatchResponse `json:"match"`
}

type MatchResponse struct {
	MatchUUID string    `json:"matchUuid"`
	MatchedAt time.Time `json:"matchedAt"`
}
