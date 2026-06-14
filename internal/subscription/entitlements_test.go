package subscription

import "testing"

func TestDecodeEntitlementsRequiresCompleteShape(t *testing.T) {
	_, err := DecodeEntitlements([]byte(`{"dailyLikeLimit":50}`))
	if err == nil {
		t.Fatal("expected incomplete entitlement JSON to fail")
	}
}

func TestDecodeEntitlementsRejectsNegativeLimits(t *testing.T) {
	_, err := DecodeEntitlements([]byte(`{
		"dailyLikeLimit": -1,
		"dailySuperLikeLimit": 1,
		"canUseAudioCall": false,
		"canUseVideoCall": false,
		"maxCallDurationSeconds": 0,
		"dailyCallLimitSeconds": 0,
		"canSeeWhoLikedMe": false,
		"canUseAdvancedFilters": false
	}`))
	if err == nil {
		t.Fatal("expected negative limit to fail")
	}
}

func TestDecodeEntitlementsRejectsNullFields(t *testing.T) {
	_, err := DecodeEntitlements([]byte(`{
		"dailyLikeLimit": 50,
		"dailySuperLikeLimit": 1,
		"canUseAudioCall": null,
		"canUseVideoCall": false,
		"maxCallDurationSeconds": 0,
		"dailyCallLimitSeconds": 0,
		"canSeeWhoLikedMe": false,
		"canUseAdvancedFilters": false
	}`))
	if err == nil {
		t.Fatal("expected null entitlement field to fail")
	}
}

func TestDecodeEntitlementsAcceptsPremiumShape(t *testing.T) {
	entitlements, err := DecodeEntitlements([]byte(`{
		"dailyLikeLimit": 300,
		"dailySuperLikeLimit": 10,
		"canUseAudioCall": true,
		"canUseVideoCall": true,
		"maxCallDurationSeconds": 1800,
		"dailyCallLimitSeconds": 7200,
		"canSeeWhoLikedMe": true,
		"canUseAdvancedFilters": true
	}`))
	if err != nil {
		t.Fatalf("expected valid entitlements: %v", err)
	}
	if entitlements.DailyLikeLimit != 300 || !entitlements.CanUseVideoCall {
		t.Fatalf("unexpected entitlements: %#v", entitlements)
	}
}

func TestFreeEntitlements(t *testing.T) {
	entitlements := FreeEntitlements()
	if entitlements.DailyLikeLimit != 50 {
		t.Fatalf("unexpected free like limit: %d", entitlements.DailyLikeLimit)
	}
	if entitlements.DailySuperLikeLimit != 1 {
		t.Fatalf("unexpected free super like limit: %d", entitlements.DailySuperLikeLimit)
	}
	if entitlements.CanUseAudioCall || entitlements.CanUseVideoCall {
		t.Fatal("free entitlements should not allow calls")
	}
}
