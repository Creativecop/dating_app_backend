package notification

import (
	"encoding/json"
	"time"
)

type UpsertFCMTokenRequest struct {
	DeviceID   string  `json:"deviceId" binding:"required"`
	FCMToken   string  `json:"fcmToken" binding:"required"`
	DeviceName *string `json:"deviceName"`
	Platform   *string `json:"platform"`
	AppVersion *string `json:"appVersion"`
	OSVersion  *string `json:"osVersion"`
}

type DeleteFCMTokenRequest struct {
	DeviceID string `json:"deviceId" binding:"required"`
}

type DeviceTokenResponse struct {
	DeviceID          string     `json:"deviceId"`
	PushEnabled       bool       `json:"pushEnabled"`
	FCMTokenUpdatedAt *time.Time `json:"fcmTokenUpdatedAt"`
}

type UpdateSettingsRequest struct {
	PushEnabled        *bool   `json:"pushEnabled"`
	NewMatchEnabled    *bool   `json:"newMatchEnabled"`
	ChatMessageEnabled *bool   `json:"chatMessageEnabled"`
	SuperLikeEnabled   *bool   `json:"superLikeEnabled"`
	QuietHoursEnabled  *bool   `json:"quietHoursEnabled"`
	QuietHoursStart    *string `json:"quietHoursStart"`
	QuietHoursEnd      *string `json:"quietHoursEnd"`
	Timezone           *string `json:"timezone"`

	present map[string]bool
	nulls   map[string]bool
}

func (r *UpdateSettingsRequest) UnmarshalJSON(data []byte) error {
	type alias UpdateSettingsRequest
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = UpdateSettingsRequest(aux)
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

func (r UpdateSettingsRequest) Has(field string) bool {
	return r.present[field]
}

func (r UpdateSettingsRequest) IsNull(field string) bool {
	return r.nulls[field]
}

type SettingsResponse struct {
	UUID               string  `json:"uuid"`
	PushEnabled        bool    `json:"pushEnabled"`
	NewMatchEnabled    bool    `json:"newMatchEnabled"`
	ChatMessageEnabled bool    `json:"chatMessageEnabled"`
	SuperLikeEnabled   bool    `json:"superLikeEnabled"`
	QuietHoursEnabled  bool    `json:"quietHoursEnabled"`
	QuietHoursStart    *string `json:"quietHoursStart"`
	QuietHoursEnd      *string `json:"quietHoursEnd"`
	Timezone           string  `json:"timezone"`
}

type NotificationListResponse struct {
	Items      []NotificationResponse `json:"items"`
	NextCursor *string                `json:"nextCursor"`
}

type NotificationResponse struct {
	NotificationUUID string         `json:"notificationUuid"`
	Type             string         `json:"type"`
	Title            string         `json:"title"`
	Body             string         `json:"body"`
	Data             map[string]any `json:"data"`
	ReadAt           *time.Time     `json:"readAt"`
	ClickedAt        *time.Time     `json:"clickedAt"`
	CreatedAt        time.Time      `json:"createdAt"`
}

type MarkReadResponse struct {
	NotificationUUID string     `json:"notificationUuid"`
	ReadAt           *time.Time `json:"readAt"`
}

type MarkAllReadResponse struct {
	Updated int64     `json:"updated"`
	ReadAt  time.Time `json:"readAt"`
}
