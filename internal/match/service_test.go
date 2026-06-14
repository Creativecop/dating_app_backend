package match

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestListCursorRoundTrip(t *testing.T) {
	original := listCursor{
		MatchedAt: time.Date(2026, 6, 14, 10, 30, 0, 0, time.UTC),
		MatchID:   120,
	}

	encoded, err := encodeListCursor(original)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	decoded, err := decodeListCursor(encoded)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if decoded == nil || *decoded != original {
		t.Fatalf("unexpected decoded cursor: %#v", decoded)
	}
}

func TestListCursorRejectsInvalidInput(t *testing.T) {
	if _, err := decodeListCursor("not-base64"); err == nil {
		t.Fatal("expected invalid cursor to fail")
	}
}

func TestNormalizeListLimitDefaultsAndClamps(t *testing.T) {
	value, err := normalizeListLimit("")
	if err != nil {
		t.Fatalf("default limit: %v", err)
	}
	if value != defaultListLimit {
		t.Fatalf("unexpected default limit: %d", value)
	}

	value, err = normalizeListLimit("1000")
	if err != nil {
		t.Fatalf("clamped limit: %v", err)
	}
	if value != maxListLimit {
		t.Fatalf("unexpected clamped limit: %d", value)
	}
}

func TestNormalizeUnmatchRequestTrimsAndBoundsReasons(t *testing.T) {
	code := "  NO_LONGER_INTERESTED  "
	note := "  no chemistry  "
	result, err := normalizeUnmatchRequest(UnmatchRequest{ReasonCode: &code, ReasonNote: &note})
	if err != nil {
		t.Fatalf("normalize unmatch: %v", err)
	}
	if result.ReasonCode == nil || *result.ReasonCode != "NO_LONGER_INTERESTED" {
		t.Fatalf("unexpected reason code: %#v", result.ReasonCode)
	}
	if result.ReasonNote == nil || *result.ReasonNote != "no chemistry" {
		t.Fatalf("unexpected reason note: %#v", result.ReasonNote)
	}

	long := strings.Repeat("x", 301)
	if _, err := normalizeUnmatchRequest(UnmatchRequest{ReasonNote: &long}); err == nil {
		t.Fatal("expected long reason note to fail")
	}
}

func TestMatchRowsSQLDoesNotDependOnDiscoveryReadiness(t *testing.T) {
	query := baseMatchRowsSQL()
	for _, forbidden := range []string{
		"discovery_eligible",
		"show_me_in_discovery",
		"user_locations",
		"discovery_preferences",
		"last_updated_at",
	} {
		if strings.Contains(query, forbidden) {
			t.Fatalf("match visibility should not depend on %q", forbidden)
		}
	}
}

func TestMatchRowListItemUsesSafePhotoURL(t *testing.T) {
	name := "Ayesha"
	bio := "Coffee and travel."
	profileUUID := "profile-uuid"
	mediaUUID := "media-uuid"
	dob := time.Date(2001, 6, 14, 0, 0, 0, 0, time.UTC)

	item := matchRow{
		MatchUUID:        "match-uuid",
		MatchedAt:        time.Date(2026, 6, 14, 10, 30, 0, 0, time.UTC),
		OtherUserUUID:    "user-uuid",
		ProfileUUID:      &profileUUID,
		DisplayName:      &name,
		DateOfBirth:      &dob,
		ShowAge:          false,
		Bio:              &bio,
		PrimaryMediaUUID: &mediaUUID,
	}.toListItem()

	if item.User.Age != nil {
		t.Fatalf("expected age to be hidden: %#v", item.User.Age)
	}
	if item.User.PrimaryPhoto == nil || item.User.PrimaryPhoto.ThumbnailURL != "/api/v1/media/media-uuid/thumbnail" {
		t.Fatalf("unexpected primary photo: %#v", item.User.PrimaryPhoto)
	}
	if item.User.PrimaryPhoto.DisplayURL != "" {
		t.Fatalf("list primary photo should not include display URL: %s", item.User.PrimaryPhoto.DisplayURL)
	}
	if !reflect.DeepEqual(item.User.ProfileUUID, &profileUUID) {
		t.Fatalf("unexpected profile uuid: %#v", item.User.ProfileUUID)
	}
}
