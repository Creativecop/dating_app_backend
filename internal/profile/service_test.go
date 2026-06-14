package profile

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gorm.io/datatypes"
)

func TestPatchProfileRequestTracksPartialFields(t *testing.T) {
	var req PatchProfileRequest
	if err := json.Unmarshal([]byte(`{"bio":"Updated bio"}`), &req); err != nil {
		t.Fatalf("unmarshal patch: %v", err)
	}
	if !req.Has("bio") {
		t.Fatal("expected bio to be marked present")
	}
	if req.Has("displayName") {
		t.Fatal("did not expect displayName to be marked present")
	}
	if err := validatePatchProfileRequest(req); err != nil {
		t.Fatalf("bio-only patch should be valid: %v", err)
	}
}

func TestPatchProfileRequestRejectsRequiredNullWhenPresent(t *testing.T) {
	var req PatchProfileRequest
	if err := json.Unmarshal([]byte(`{"displayName":null}`), &req); err != nil {
		t.Fatalf("unmarshal patch: %v", err)
	}
	if !req.Has("displayName") || !req.IsNull("displayName") {
		t.Fatal("expected displayName null to be tracked")
	}
	if err := validatePatchProfileRequest(req); err == nil {
		t.Fatal("expected null displayName to be rejected")
	}
}

func TestAgeValidationUsesUTCDateOnly(t *testing.T) {
	now := time.Date(2026, 6, 14, 23, 30, 0, 0, time.UTC)
	adult := time.Date(2008, 6, 14, 0, 0, 0, 0, time.UTC)
	underAge := time.Date(2008, 6, 15, 0, 0, 0, 0, time.UTC)

	if !IsAdultDOB(adult, now) {
		t.Fatal("expected exact 18th birthday to pass")
	}
	if IsAdultDOB(underAge, now) {
		t.Fatal("expected one day under 18 to fail")
	}
}

func TestCompletionMissing(t *testing.T) {
	name := "Tanvir"
	gender := GenderMale
	lookingFor := LookingForFemale
	dob := time.Date(1998, 5, 20, 0, 0, 0, 0, time.UTC)

	profile := Profile{
		DisplayName:      &name,
		DateOfBirth:      &dob,
		Gender:           &gender,
		LookingForGender: &lookingFor,
		ProfileStatus:    ProfileStatusDraft,
	}

	missing := completionMissing(profile, 2, 1, time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC))
	if strings.Join(missing, ",") != "interests,prompts" {
		t.Fatalf("unexpected missing fields: %#v", missing)
	}
}

func TestNormalizeLifestyleAnswerRejectsDuplicateMultipleChoiceValues(t *testing.T) {
	question := LifestyleQuestion{
		AnswerType: AnswerTypeMultipleChoice,
		Options:    datatypes.JSON([]byte(`["A","B","C"]`)),
	}

	_, err := normalizeLifestyleAnswer(question, json.RawMessage(`["A","A"]`))
	if err == nil {
		t.Fatal("expected duplicate multiple choice values to fail")
	}
}

func TestNormalizeLifestyleAnswerValidatesSingleChoiceOptions(t *testing.T) {
	question := LifestyleQuestion{
		AnswerType: AnswerTypeSingleChoice,
		Options:    datatypes.JSON([]byte(`["NO","YES"]`)),
	}

	value, err := normalizeLifestyleAnswer(question, json.RawMessage(`"NO"`))
	if err != nil {
		t.Fatalf("expected valid single choice: %v", err)
	}
	if string(value) != `"NO"` {
		t.Fatalf("unexpected normalized value: %s", string(value))
	}

	if _, err := normalizeLifestyleAnswer(question, json.RawMessage(`"MAYBE"`)); err == nil {
		t.Fatal("expected invalid option to fail")
	}
}
