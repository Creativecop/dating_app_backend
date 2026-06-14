package discovery

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNormalizePreferencesRequiresFullReplacementFields(t *testing.T) {
	var req UpdatePreferencesRequest
	if err := json.Unmarshal([]byte(`{"minAge":18}`), &req); err != nil {
		t.Fatalf("unmarshal preferences: %v", err)
	}

	if _, err := normalizePreferencesRequest(req); err == nil {
		t.Fatal("expected missing full replacement fields to fail")
	}
}

func TestNormalizePreferencesNormalizesEveryone(t *testing.T) {
	var req UpdatePreferencesRequest
	if err := json.Unmarshal([]byte(`{
		"minAge": 22,
		"maxAge": 35,
		"preferredGenders": ["FEMALE", "EVERYONE"],
		"maxDistanceKm": 30,
		"verifiedOnly": false,
		"showMeInDiscovery": true,
		"hideDistance": false
	}`), &req); err != nil {
		t.Fatalf("unmarshal preferences: %v", err)
	}

	result, err := normalizePreferencesRequest(req)
	if err != nil {
		t.Fatalf("expected valid preferences: %v", err)
	}
	if !reflect.DeepEqual(result.PreferredGenders, []string{GenderEveryone}) {
		t.Fatalf("unexpected genders: %#v", result.PreferredGenders)
	}
}

func TestNormalizePreferencesRejectsDuplicateGenders(t *testing.T) {
	var req UpdatePreferencesRequest
	if err := json.Unmarshal([]byte(`{
		"minAge": 22,
		"maxAge": 35,
		"preferredGenders": ["FEMALE", "female"],
		"maxDistanceKm": 30,
		"verifiedOnly": false,
		"showMeInDiscovery": true,
		"hideDistance": false
	}`), &req); err != nil {
		t.Fatalf("unmarshal preferences: %v", err)
	}

	if _, err := normalizePreferencesRequest(req); err == nil {
		t.Fatal("expected duplicate genders to fail")
	}
}

func TestBuildReadinessSplitsMissingAndBlocked(t *testing.T) {
	response := buildReadiness(readinessSnapshot{
		UserStatus:                  UserStatusActive,
		HasProfile:                  true,
		ProfileStatus:               ProfileStatusActive,
		HasApprovedPrimaryPhoto:     false,
		HasLocation:                 true,
		HasPreciseConsentedLocation: true,
		HasFreshLocation:            true,
		HasPreferences:              true,
		ShowMeInDiscovery:           false,
	})

	if response.DiscoveryEligible {
		t.Fatal("expected readiness to be false")
	}
	if !reflect.DeepEqual(response.Missing, []string{"approvedPrimaryPhoto"}) {
		t.Fatalf("unexpected missing reasons: %#v", response.Missing)
	}
	if !reflect.DeepEqual(response.Blocked, []string{"showMeInDiscoveryDisabled"}) {
		t.Fatalf("unexpected blocked reasons: %#v", response.Blocked)
	}
}

func TestReadinessErrorDetailsUsesFrontendFriendlyKeys(t *testing.T) {
	details := readinessErrorDetails(&ReadinessResponse{
		Missing: []string{"approvedPrimaryPhoto", "freshLocation", "preferences"},
		Blocked: []string{"showMeInDiscoveryDisabled", "profileUnderReview"},
	})

	expectedMissing := []string{"primary_photo", "fresh_location", "discovery_preferences"}
	expectedBlocked := []string{"show_me_in_discovery_disabled", "profile_under_review"}
	if !reflect.DeepEqual(details["missing"], expectedMissing) {
		t.Fatalf("unexpected missing details: %#v", details["missing"])
	}
	if !reflect.DeepEqual(details["blocked"], expectedBlocked) {
		t.Fatalf("unexpected blocked details: %#v", details["blocked"])
	}
}

func TestFeedCursorRoundTrip(t *testing.T) {
	original := feedCursor{
		DistanceMeters:  4200,
		CompletedAt:     time.Date(2026, 6, 14, 10, 30, 0, 0, time.UTC),
		CandidateUserID: 123,
	}

	encoded, err := encodeFeedCursor(original)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	decoded, err := decodeFeedCursor(encoded)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if decoded == nil || *decoded != original {
		t.Fatalf("unexpected decoded cursor: %#v", decoded)
	}
}

func TestFeedCursorRejectsInvalidInput(t *testing.T) {
	if _, err := decodeFeedCursor("not-base64"); err == nil {
		t.Fatal("expected invalid cursor to fail")
	}
}

func TestNormalizeFeedLimitDefaultsAndClamps(t *testing.T) {
	value, err := normalizeFeedLimit("")
	if err != nil {
		t.Fatalf("default limit: %v", err)
	}
	if value != defaultFeedLimit {
		t.Fatalf("unexpected default limit: %d", value)
	}

	value, err = normalizeFeedLimit("1000")
	if err != nil {
		t.Fatalf("clamped limit: %v", err)
	}
	if value != maxFeedLimit {
		t.Fatalf("unexpected clamped limit: %d", value)
	}
}

func TestFeedRowPreviewRespectsPrivacyFields(t *testing.T) {
	name := "Ayesha"
	bio := "Coffee and travel."
	dob := time.Date(2001, 6, 14, 0, 0, 0, 0, time.UTC)

	preview, err := feedRow{
		CandidateUserUUID: "user-uuid",
		ProfileUUID:       "profile-uuid",
		DisplayName:       &name,
		DateOfBirth:       &dob,
		ShowAge:           false,
		Bio:               &bio,
		HideDistance:      true,
		DistanceMeters:    4200,
		PrimaryMediaUUID:  "media-uuid",
		InterestsJSON:     `["Travel","Coffee"]`,
		PromptsJSON:       `[{"question":"Weekend?","answer":"Road trip."}]`,
	}.toPreview()
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Age != nil {
		t.Fatalf("expected age to be hidden: %#v", preview.Age)
	}
	if preview.DistanceKM != nil || !preview.DistanceHidden {
		t.Fatalf("expected distance to be hidden: km=%#v hidden=%v", preview.DistanceKM, preview.DistanceHidden)
	}
	if preview.PrimaryPhoto.DisplayURL != "/api/v1/media/media-uuid/display" {
		t.Fatalf("unexpected display URL: %s", preview.PrimaryPhoto.DisplayURL)
	}
	if !reflect.DeepEqual(preview.Interests, []string{"Travel", "Coffee"}) {
		t.Fatalf("unexpected interests: %#v", preview.Interests)
	}
	if len(preview.Prompts) != 1 || preview.Prompts[0].Question != "Weekend?" {
		t.Fatalf("unexpected prompts: %#v", preview.Prompts)
	}
}

func TestNormalizeActionRequestRequiresValidClientActionID(t *testing.T) {
	_, err := normalizeActionRequest(CreateActionRequest{
		TargetUserUUID: "0f6c87d9-8fa3-44d4-9d50-103091b69b0d",
		ActionType:     ActionLike,
		ClientActionID: "",
	})
	if err == nil {
		t.Fatal("expected missing clientActionId to fail")
	}

	result, err := normalizeActionRequest(CreateActionRequest{
		TargetUserUUID: "0f6c87d9-8fa3-44d4-9d50-103091b69b0d",
		ActionType:     "super_like",
		ClientActionID: "3171becc-62a7-4e86-9e6e-d0fbbad88534",
	})
	if err != nil {
		t.Fatalf("expected valid action request: %v", err)
	}
	if result.ActionType != ActionSuperLike {
		t.Fatalf("unexpected action type: %s", result.ActionType)
	}
}

func TestOrderedPairAndUTCDate(t *testing.T) {
	low, high := orderedPair(20, 10)
	if low != 10 || high != 20 {
		t.Fatalf("unexpected ordered pair: %d %d", low, high)
	}

	value := utcDate(time.Date(2026, 6, 14, 23, 59, 0, 0, time.FixedZone("BDT", 6*60*60)))
	expected := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	if !value.Equal(expected) {
		t.Fatalf("unexpected UTC date: %s", value)
	}
}

func TestDiscoverabilitySQLExcludesActionsOnlyForFeed(t *testing.T) {
	feedSQL := baseDiscoverableCandidatesSQL(DiscoverabilityModeFeed)
	actionSQL := baseDiscoverableCandidatesSQL(DiscoverabilityModeAction)

	if !strings.Contains(feedSQL, "FROM discovery_actions da") {
		t.Fatal("expected feed SQL to exclude acted users")
	}
	if strings.Contains(actionSQL, "FROM discovery_actions da") {
		t.Fatal("did not expect action SQL to exclude acted users")
	}
}
