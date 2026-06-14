package subscription

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	FeatureLikeDaily             = "LIKE_DAILY"
	FeatureSuperLikeDaily        = "SUPER_LIKE_DAILY"
	FeatureAudioCallDailySeconds = "AUDIO_CALL_DAILY_SECONDS"
	FeatureVideoCallDailySeconds = "VIDEO_CALL_DAILY_SECONDS"
)

type Entitlements struct {
	DailyLikeLimit         int  `json:"dailyLikeLimit"`
	DailySuperLikeLimit    int  `json:"dailySuperLikeLimit"`
	CanUseAudioCall        bool `json:"canUseAudioCall"`
	CanUseVideoCall        bool `json:"canUseVideoCall"`
	MaxCallDurationSeconds int  `json:"maxCallDurationSeconds"`
	DailyCallLimitSeconds  int  `json:"dailyCallLimitSeconds"`
	CanSeeWhoLikedMe       bool `json:"canSeeWhoLikedMe"`
	CanUseAdvancedFilters  bool `json:"canUseAdvancedFilters"`
}

func FreeEntitlements() Entitlements {
	return Entitlements{
		DailyLikeLimit:         50,
		DailySuperLikeLimit:    1,
		CanUseAudioCall:        false,
		CanUseVideoCall:        false,
		MaxCallDurationSeconds: 0,
		DailyCallLimitSeconds:  0,
		CanSeeWhoLikedMe:       false,
		CanUseAdvancedFilters:  false,
	}
}

func DecodeEntitlements(raw []byte) (Entitlements, error) {
	required := []string{
		"dailyLikeLimit",
		"dailySuperLikeLimit",
		"canUseAudioCall",
		"canUseVideoCall",
		"maxCallDurationSeconds",
		"dailyCallLimitSeconds",
		"canSeeWhoLikedMe",
		"canUseAdvancedFilters",
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return Entitlements{}, fmt.Errorf("invalid entitlements JSON: %w", err)
	}
	for _, field := range required {
		if _, ok := fields[field]; !ok {
			return Entitlements{}, fmt.Errorf("missing entitlement field %s", field)
		}
	}
	entitlements := Entitlements{}
	var err error
	if entitlements.DailyLikeLimit, err = decodeRequiredInt(fields, "dailyLikeLimit"); err != nil {
		return Entitlements{}, err
	}
	if entitlements.DailySuperLikeLimit, err = decodeRequiredInt(fields, "dailySuperLikeLimit"); err != nil {
		return Entitlements{}, err
	}
	if entitlements.CanUseAudioCall, err = decodeRequiredBool(fields, "canUseAudioCall"); err != nil {
		return Entitlements{}, err
	}
	if entitlements.CanUseVideoCall, err = decodeRequiredBool(fields, "canUseVideoCall"); err != nil {
		return Entitlements{}, err
	}
	if entitlements.MaxCallDurationSeconds, err = decodeRequiredInt(fields, "maxCallDurationSeconds"); err != nil {
		return Entitlements{}, err
	}
	if entitlements.DailyCallLimitSeconds, err = decodeRequiredInt(fields, "dailyCallLimitSeconds"); err != nil {
		return Entitlements{}, err
	}
	if entitlements.CanSeeWhoLikedMe, err = decodeRequiredBool(fields, "canSeeWhoLikedMe"); err != nil {
		return Entitlements{}, err
	}
	if entitlements.CanUseAdvancedFilters, err = decodeRequiredBool(fields, "canUseAdvancedFilters"); err != nil {
		return Entitlements{}, err
	}
	if err := ValidateEntitlements(entitlements); err != nil {
		return Entitlements{}, err
	}
	return entitlements, nil
}

func ValidateEntitlements(entitlements Entitlements) error {
	if entitlements.DailyLikeLimit < 0 {
		return fmt.Errorf("dailyLikeLimit cannot be negative")
	}
	if entitlements.DailySuperLikeLimit < 0 {
		return fmt.Errorf("dailySuperLikeLimit cannot be negative")
	}
	if entitlements.MaxCallDurationSeconds < 0 {
		return fmt.Errorf("maxCallDurationSeconds cannot be negative")
	}
	if entitlements.DailyCallLimitSeconds < 0 {
		return fmt.Errorf("dailyCallLimitSeconds cannot be negative")
	}
	return nil
}

func decodeRequiredInt(fields map[string]json.RawMessage, name string) (int, error) {
	raw := fields[name]
	if strings.TrimSpace(string(raw)) == "null" {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return value, nil
}

func decodeRequiredBool(fields map[string]json.RawMessage, name string) (bool, error) {
	raw := fields[name]
	if strings.TrimSpace(string(raw)) == "null" {
		return false, fmt.Errorf("%s must be a boolean", name)
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, fmt.Errorf("%s must be a boolean", name)
	}
	return value, nil
}
