package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/neoscoder/aura-backend/internal/config"
)

const dateLayout = "2006-01-02"

type Service struct {
	db                          *gorm.DB
	repo                        *Repository
	discoveryLocationMaxAgeDays int
}

func NewService(db *gorm.DB, discoveryCfg ...config.DiscoveryConfig) *Service {
	maxAgeDays := 30
	if len(discoveryCfg) > 0 && discoveryCfg[0].LocationMaxAgeDays > 0 {
		maxAgeDays = discoveryCfg[0].LocationMaxAgeDays
	}
	return &Service{
		db:                          db,
		repo:                        NewRepository(db),
		discoveryLocationMaxAgeDays: maxAgeDays,
	}
}

func (s *Service) EnsureProfile(ctx context.Context, userID uint64) error {
	return s.repo.EnsureProfile(ctx, userID)
}

func (s *Service) RefreshDiscoveryEligibility(ctx context.Context, userID uint64) error {
	var eligible bool
	freshSince := time.Now().UTC().AddDate(0, 0, -s.discoveryLocationMaxAgeDays)
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM users u
			JOIN profiles p ON p.user_id = u.id
			JOIN discovery_preferences dp ON dp.user_id = u.id
			JOIN user_locations ul ON ul.user_id = u.id
			WHERE u.id = ?
			  AND u.status = 'ACTIVE'
			  AND u.deleted_at IS NULL
			  AND p.profile_status = 'ACTIVE'
			  AND dp.show_me_in_discovery = TRUE
			  AND ul.source IN ('GPS', 'MANUAL')
			  AND ul.is_precise = TRUE
			  AND ul.location_consent_at IS NOT NULL
			  AND ul.last_updated_at >= ?
			  AND EXISTS (
			    SELECT 1
			    FROM user_media m
			    WHERE m.user_id = u.id
			      AND m.media_purpose = 'PROFILE_PHOTO'
			      AND m.processing_status = 'READY'
			      AND m.moderation_status = 'APPROVED'
			      AND m.is_primary = TRUE
			      AND m.deleted_at IS NULL
			  )
		)
	`, userID, freshSince).Scan(&eligible).Error
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&Profile{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"discovery_eligible":               eligible,
			"discovery_eligibility_updated_at": now,
			"updated_at":                       now,
		}).Error
}

func (s *Service) GetMe(ctx context.Context, userID uint64) (*ProfileMeResponse, error) {
	if err := s.EnsureProfile(ctx, userID); err != nil {
		return nil, err
	}

	profile, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, validationError("Profile could not be created", nil)
	}

	completion, err := s.recalculateCompletion(ctx, s.db, profile)
	if err != nil {
		return nil, err
	}

	return s.buildProfileMe(ctx, *profile, completion)
}

func (s *Service) UpdateProfile(ctx context.Context, userID uint64, req PatchProfileRequest) (*ProfileMeResponse, error) {
	if err := validatePatchProfileRequest(req); err != nil {
		return nil, err
	}

	var profile Profile
	var completion CompletionResponse
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.WithDB(tx).EnsureProfile(ctx, userID); err != nil {
			return err
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).
			First(&profile).Error; err != nil {
			return err
		}

		updates, err := patchUpdates(req)
		if err != nil {
			return err
		}
		if len(updates) > 0 {
			updates["updated_at"] = time.Now().UTC()
			if err := tx.Model(&Profile{}).Where("id = ?", profile.ID).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Where("id = ?", profile.ID).First(&profile).Error; err != nil {
				return err
			}
		}

		nextCompletion, err := s.recalculateCompletion(ctx, tx, &profile)
		if err != nil {
			return err
		}
		if profile.ProfileStatus == ProfileStatusActive && len(nextCompletion.Missing) > 0 {
			return incompleteError(nextCompletion.Missing)
		}
		completion = nextCompletion
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.RefreshDiscoveryEligibility(ctx, userID); err != nil {
		return nil, err
	}

	return s.buildProfileMe(ctx, profile, completion)
}

func (s *Service) UpdateInterests(ctx context.Context, userID uint64, req UpdateInterestsRequest) (*ProfileMeResponse, error) {
	ids, err := validateUniqueIDs(req.InterestIDs, 3, 15, "interestIds")
	if err != nil {
		return nil, err
	}

	var profile Profile
	var completion CompletionResponse
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.WithDB(tx).EnsureProfile(ctx, userID); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&profile).Error; err != nil {
			return err
		}

		var interests []Interest
		if err := tx.Where("id IN ? AND is_active = ?", ids, true).Find(&interests).Error; err != nil {
			return err
		}
		if len(interests) != len(ids) {
			return validationError("One or more interests are invalid", map[string]any{"field": "interestIds"})
		}

		if err := tx.Where("user_id = ?", userID).Delete(&UserInterest{}).Error; err != nil {
			return err
		}

		now := time.Now().UTC()
		rows := make([]UserInterest, 0, len(ids))
		for _, id := range ids {
			rows = append(rows, UserInterest{
				UUID:       uuid.New(),
				UserID:     userID,
				InterestID: id,
				CreatedAt:  now,
			})
		}
		if len(rows) > 0 {
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
		}

		nextCompletion, err := s.recalculateCompletion(ctx, tx, &profile)
		if err != nil {
			return err
		}
		if profile.ProfileStatus == ProfileStatusActive && len(nextCompletion.Missing) > 0 {
			return incompleteError(nextCompletion.Missing)
		}
		completion = nextCompletion
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.RefreshDiscoveryEligibility(ctx, userID); err != nil {
		return nil, err
	}

	return s.buildProfileMe(ctx, profile, completion)
}

func (s *Service) UpdatePrompts(ctx context.Context, userID uint64, req UpdatePromptsRequest) (*ProfileMeResponse, error) {
	if len(req.Prompts) < 2 || len(req.Prompts) > 3 {
		return nil, validationError("Select 2 to 3 profile prompts", map[string]any{"field": "prompts"})
	}

	ids := make([]uint64, 0, len(req.Prompts))
	answers := make(map[uint64]string, len(req.Prompts))
	seen := make(map[uint64]bool, len(req.Prompts))
	for _, item := range req.Prompts {
		if item.PromptQuestionID == 0 {
			return nil, validationError("Prompt question is required", map[string]any{"field": "promptQuestionId"})
		}
		if seen[item.PromptQuestionID] {
			return nil, validationError("Prompt question IDs must be unique", map[string]any{"field": "prompts"})
		}
		seen[item.PromptQuestionID] = true

		answer := strings.TrimSpace(item.Answer)
		if answer == "" || len(answer) > 300 {
			return nil, validationError("Prompt answers must be 1 to 300 characters", map[string]any{"field": "answer"})
		}
		ids = append(ids, item.PromptQuestionID)
		answers[item.PromptQuestionID] = answer
	}

	var profile Profile
	var completion CompletionResponse
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.WithDB(tx).EnsureProfile(ctx, userID); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&profile).Error; err != nil {
			return err
		}

		var questions []ProfilePromptQuestion
		if err := tx.Where("id IN ? AND is_active = ?", ids, true).Find(&questions).Error; err != nil {
			return err
		}
		if len(questions) != len(ids) {
			return validationError("One or more prompt questions are invalid", map[string]any{"field": "prompts"})
		}

		if err := tx.Where("user_id = ?", userID).Delete(&UserProfilePrompt{}).Error; err != nil {
			return err
		}

		now := time.Now().UTC()
		rows := make([]UserProfilePrompt, 0, len(ids))
		for _, id := range ids {
			rows = append(rows, UserProfilePrompt{
				UUID:             uuid.New(),
				UserID:           userID,
				PromptQuestionID: id,
				Answer:           answers[id],
				CreatedAt:        now,
				UpdatedAt:        now,
			})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}

		nextCompletion, err := s.recalculateCompletion(ctx, tx, &profile)
		if err != nil {
			return err
		}
		if profile.ProfileStatus == ProfileStatusActive && len(nextCompletion.Missing) > 0 {
			return incompleteError(nextCompletion.Missing)
		}
		completion = nextCompletion
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.buildProfileMe(ctx, profile, completion)
}

func (s *Service) UpdateLifestyle(ctx context.Context, userID uint64, req UpdateLifestyleRequest) (*ProfileMeResponse, error) {
	ids := make([]uint64, 0, len(req.Answers))
	seen := make(map[uint64]bool, len(req.Answers))
	for _, item := range req.Answers {
		if item.LifestyleQuestionID == 0 {
			return nil, validationError("Lifestyle question is required", map[string]any{"field": "lifestyleQuestionId"})
		}
		if seen[item.LifestyleQuestionID] {
			return nil, validationError("Lifestyle question IDs must be unique", map[string]any{"field": "answers"})
		}
		seen[item.LifestyleQuestionID] = true
		ids = append(ids, item.LifestyleQuestionID)
	}

	var profile Profile
	var completion CompletionResponse
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.WithDB(tx).EnsureProfile(ctx, userID); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&profile).Error; err != nil {
			return err
		}

		questions := make(map[uint64]LifestyleQuestion, len(ids))
		if len(ids) > 0 {
			var rows []LifestyleQuestion
			if err := tx.Where("id IN ? AND is_active = ?", ids, true).Find(&rows).Error; err != nil {
				return err
			}
			if len(rows) != len(ids) {
				return validationError("One or more lifestyle questions are invalid", map[string]any{"field": "answers"})
			}
			for _, row := range rows {
				questions[row.ID] = row
			}
		}

		if err := tx.Where("user_id = ?", userID).Delete(&UserLifestyleAnswer{}).Error; err != nil {
			return err
		}

		now := time.Now().UTC()
		rows := make([]UserLifestyleAnswer, 0, len(req.Answers))
		for _, item := range req.Answers {
			question := questions[item.LifestyleQuestionID]
			normalized, err := normalizeLifestyleAnswer(question, item.Answer)
			if err != nil {
				return err
			}
			rows = append(rows, UserLifestyleAnswer{
				UUID:                uuid.New(),
				UserID:              userID,
				LifestyleQuestionID: item.LifestyleQuestionID,
				Answer:              normalized,
				CreatedAt:           now,
				UpdatedAt:           now,
			})
		}
		if len(rows) > 0 {
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
		}

		nextCompletion, err := s.recalculateCompletion(ctx, tx, &profile)
		if err != nil {
			return err
		}
		if profile.ProfileStatus == ProfileStatusActive && len(nextCompletion.Missing) > 0 {
			return incompleteError(nextCompletion.Missing)
		}
		completion = nextCompletion
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.buildProfileMe(ctx, profile, completion)
}

func (s *Service) CompleteProfile(ctx context.Context, userID uint64) (*ProfileMeResponse, error) {
	var profile Profile
	completion := CompletionResponse{}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.WithDB(tx).EnsureProfile(ctx, userID); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&profile).Error; err != nil {
			return err
		}

		nextCompletion, err := s.computeCompletion(ctx, tx, profile)
		if err != nil {
			return err
		}
		if len(nextCompletion.Missing) > 0 {
			return incompleteError(nextCompletion.Missing)
		}

		now := time.Now().UTC()
		if err := tx.Model(&Profile{}).
			Where("id = ?", profile.ID).
			Updates(map[string]any{
				"profile_status":        ProfileStatusActive,
				"completion_percentage": 100,
				"completed_at":          now,
				"updated_at":            now,
			}).Error; err != nil {
			return err
		}
		if err := tx.Table("users").
			Where("id = ? AND deleted_at IS NULL", userID).
			Updates(map[string]any{
				"onboarding_status": "COMPLETED",
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", profile.ID).First(&profile).Error; err != nil {
			return err
		}
		completion = CompletionResponse{Percentage: 100, Status: ProfileStatusActive, Missing: []string{}}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.RefreshDiscoveryEligibility(ctx, userID); err != nil {
		return nil, err
	}

	return s.buildProfileMe(ctx, profile, completion)
}

func (s *Service) InterestCatalog(ctx context.Context) (*InterestCatalogResponse, error) {
	categories, err := s.repo.ListInterestCategories(ctx)
	if err != nil {
		return nil, err
	}
	response := &InterestCatalogResponse{Categories: make([]InterestCategoryResponse, 0, len(categories))}
	for _, category := range categories {
		item := InterestCategoryResponse{
			ID:          category.ID,
			UUID:        category.UUID.String(),
			CategoryKey: category.CategoryKey,
			Name:        category.Name,
			SortOrder:   category.SortOrder,
			Interests:   make([]InterestResponse, 0, len(category.Interests)),
		}
		for _, interest := range category.Interests {
			item.Interests = append(item.Interests, toInterestResponse(interest))
		}
		response.Categories = append(response.Categories, item)
	}
	return response, nil
}

func (s *Service) PromptCatalog(ctx context.Context) ([]PromptQuestionResponse, error) {
	questions, err := s.repo.ListPromptQuestions(ctx)
	if err != nil {
		return nil, err
	}
	response := make([]PromptQuestionResponse, 0, len(questions))
	for _, question := range questions {
		response = append(response, toPromptQuestionResponse(question))
	}
	return response, nil
}

func (s *Service) LifestyleCatalog(ctx context.Context) ([]LifestyleQuestionResponse, error) {
	questions, err := s.repo.ListLifestyleQuestions(ctx)
	if err != nil {
		return nil, err
	}
	response := make([]LifestyleQuestionResponse, 0, len(questions))
	for _, question := range questions {
		item, err := toLifestyleQuestionResponse(question)
		if err != nil {
			return nil, err
		}
		response = append(response, item)
	}
	return response, nil
}

func (s *Service) buildProfileMe(ctx context.Context, profile Profile, completion CompletionResponse) (*ProfileMeResponse, error) {
	if completion.Status == "" {
		var err error
		completion, err = s.recalculateCompletion(ctx, s.db, &profile)
		if err != nil {
			return nil, err
		}
	}

	interests, err := s.repo.UserInterests(ctx, profile.UserID)
	if err != nil {
		return nil, err
	}
	prompts, err := s.repo.UserPrompts(ctx, profile.UserID)
	if err != nil {
		return nil, err
	}
	lifestyle, err := s.repo.UserLifestyleAnswers(ctx, profile.UserID)
	if err != nil {
		return nil, err
	}

	response := &ProfileMeResponse{
		Profile:          toProfileResponse(profile, completion),
		Interests:        make([]InterestResponse, 0, len(interests)),
		Prompts:          make([]UserPromptResponse, 0, len(prompts)),
		LifestyleAnswers: make([]LifestyleAnswerResponse, 0, len(lifestyle)),
		Completion:       completion,
	}

	for _, row := range interests {
		response.Interests = append(response.Interests, toInterestResponse(row.Interest))
	}
	for _, row := range prompts {
		response.Prompts = append(response.Prompts, UserPromptResponse{
			PromptQuestion: toPromptQuestionResponse(row.Question),
			Answer:         row.Answer,
		})
	}
	for _, row := range lifestyle {
		question, err := toLifestyleQuestionResponse(row.Question)
		if err != nil {
			return nil, err
		}
		answer, err := unmarshalJSONAny(row.Answer)
		if err != nil {
			return nil, err
		}
		response.LifestyleAnswers = append(response.LifestyleAnswers, LifestyleAnswerResponse{
			LifestyleQuestion: question,
			Answer:            answer,
		})
	}

	return response, nil
}

func (s *Service) recalculateCompletion(ctx context.Context, tx *gorm.DB, profile *Profile) (CompletionResponse, error) {
	completion, err := s.computeCompletion(ctx, tx, *profile)
	if err != nil {
		return CompletionResponse{}, err
	}
	if profile.CompletionPercentage != completion.Percentage {
		now := time.Now().UTC()
		if err := tx.WithContext(ctx).Model(&Profile{}).
			Where("id = ?", profile.ID).
			Updates(map[string]any{
				"completion_percentage": completion.Percentage,
				"updated_at":            now,
			}).Error; err != nil {
			return CompletionResponse{}, err
		}
		profile.CompletionPercentage = completion.Percentage
		profile.UpdatedAt = now
	}
	return completion, nil
}

func (s *Service) computeCompletion(ctx context.Context, tx *gorm.DB, profile Profile) (CompletionResponse, error) {
	interestCount, err := s.repo.CountActiveInterests(ctx, tx, profile.UserID)
	if err != nil {
		return CompletionResponse{}, err
	}
	promptCount, err := s.repo.CountActivePrompts(ctx, tx, profile.UserID)
	if err != nil {
		return CompletionResponse{}, err
	}

	missing := completionMissing(profile, interestCount, promptCount, time.Now().UTC())
	done := 6 - len(missing)
	percentage := (done * 100) / 6
	if len(missing) == 0 {
		if profile.ProfileStatus == ProfileStatusActive {
			percentage = 100
		} else {
			percentage = 95
		}
	}

	return CompletionResponse{
		Percentage: percentage,
		Status:     profile.ProfileStatus,
		Missing:    missing,
	}, nil
}

func completionMissing(profile Profile, interestCount int64, promptCount int64, now time.Time) []string {
	missing := make([]string, 0, 6)
	if profile.DisplayName == nil || strings.TrimSpace(*profile.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if profile.DateOfBirth == nil || !IsAdultDOB(*profile.DateOfBirth, now) {
		missing = append(missing, "dateOfBirth")
	}
	if profile.Gender == nil || !validGender(*profile.Gender) {
		missing = append(missing, "gender")
	}
	if profile.LookingForGender == nil || !validLookingForGender(*profile.LookingForGender) {
		missing = append(missing, "lookingForGender")
	}
	if interestCount < 3 {
		missing = append(missing, "interests")
	}
	if promptCount < 2 {
		missing = append(missing, "prompts")
	}
	return missing
}

func validatePatchProfileRequest(req PatchProfileRequest) error {
	if req.Has("displayName") && (req.DisplayName == nil || strings.TrimSpace(*req.DisplayName) == "" || len(strings.TrimSpace(*req.DisplayName)) > 120) {
		return validationError("Display name must be 1 to 120 characters", map[string]any{"field": "displayName"})
	}
	if req.Has("dateOfBirth") {
		if req.DateOfBirth == nil {
			return validationError("Date of birth is required", map[string]any{"field": "dateOfBirth"})
		}
		dob, err := ParseDateOfBirth(*req.DateOfBirth)
		if err != nil {
			return validationError("Date of birth must use YYYY-MM-DD", map[string]any{"field": "dateOfBirth"})
		}
		if !IsAdultDOB(dob, time.Now().UTC()) {
			return validationError("User must be at least 18 years old", map[string]any{"field": "dateOfBirth"})
		}
	}
	if req.Has("gender") && (req.Gender == nil || !validGender(*req.Gender)) {
		return validationError("Gender is invalid", map[string]any{"field": "gender"})
	}
	if req.Has("lookingForGender") && (req.LookingForGender == nil || !validLookingForGender(*req.LookingForGender)) {
		return validationError("Looking for gender is invalid", map[string]any{"field": "lookingForGender"})
	}
	if req.Has("bio") && req.Bio != nil && len(strings.TrimSpace(*req.Bio)) > 500 {
		return validationError("Bio must be 500 characters or less", map[string]any{"field": "bio"})
	}
	if req.Has("heightCm") && req.HeightCM != nil && (*req.HeightCM < 100 || *req.HeightCM > 250) {
		return validationError("Height must be between 100 and 250 cm", map[string]any{"field": "heightCm"})
	}
	if req.Has("relationshipGoal") && req.RelationshipGoal != nil && !validRelationshipGoal(*req.RelationshipGoal) {
		return validationError("Relationship goal is invalid", map[string]any{"field": "relationshipGoal"})
	}
	if req.Has("showAge") && req.ShowAge == nil {
		return validationError("showAge cannot be null", map[string]any{"field": "showAge"})
	}
	if req.Has("showDistance") && req.ShowDistance == nil {
		return validationError("showDistance cannot be null", map[string]any{"field": "showDistance"})
	}

	stringMax := map[string]struct {
		value *string
		max   int
	}{
		"education": {req.Education, 150},
		"jobTitle":  {req.JobTitle, 150},
		"company":   {req.Company, 150},
		"city":      {req.City, 120},
		"country":   {req.Country, 120},
	}
	for field, item := range stringMax {
		if req.Has(field) && item.value != nil && len(strings.TrimSpace(*item.value)) > item.max {
			return validationError(fmt.Sprintf("%s is too long", field), map[string]any{"field": field})
		}
	}

	return nil
}

func patchUpdates(req PatchProfileRequest) (map[string]any, error) {
	updates := make(map[string]any)
	if req.Has("displayName") {
		updates["display_name"] = strings.TrimSpace(*req.DisplayName)
	}
	if req.Has("dateOfBirth") {
		dob, err := ParseDateOfBirth(*req.DateOfBirth)
		if err != nil {
			return nil, err
		}
		updates["date_of_birth"] = dob
	}
	if req.Has("gender") {
		updates["gender"] = strings.ToUpper(strings.TrimSpace(*req.Gender))
	}
	if req.Has("lookingForGender") {
		updates["looking_for_gender"] = strings.ToUpper(strings.TrimSpace(*req.LookingForGender))
	}
	if req.Has("bio") {
		updates["bio"] = optionalString(req.Bio)
	}
	if req.Has("heightCm") {
		if req.HeightCM == nil {
			updates["height_cm"] = nil
		} else {
			updates["height_cm"] = *req.HeightCM
		}
	}
	if req.Has("education") {
		updates["education"] = optionalString(req.Education)
	}
	if req.Has("jobTitle") {
		updates["job_title"] = optionalString(req.JobTitle)
	}
	if req.Has("company") {
		updates["company"] = optionalString(req.Company)
	}
	if req.Has("city") {
		updates["city"] = optionalString(req.City)
	}
	if req.Has("country") {
		updates["country"] = optionalString(req.Country)
	}
	if req.Has("relationshipGoal") {
		if req.RelationshipGoal == nil {
			updates["relationship_goal"] = nil
		} else {
			updates["relationship_goal"] = strings.ToUpper(strings.TrimSpace(*req.RelationshipGoal))
		}
	}
	if req.Has("showAge") {
		updates["show_age"] = *req.ShowAge
	}
	if req.Has("showDistance") {
		updates["show_distance"] = *req.ShowDistance
	}
	return updates, nil
}

func validateUniqueIDs(ids []uint64, min int, max int, field string) ([]uint64, error) {
	if len(ids) < min || len(ids) > max {
		return nil, validationError(fmt.Sprintf("%s must contain %d to %d IDs", field, min, max), map[string]any{"field": field})
	}
	seen := make(map[uint64]bool, len(ids))
	unique := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			return nil, validationError(fmt.Sprintf("%s contains an invalid ID", field), map[string]any{"field": field})
		}
		if seen[id] {
			return nil, validationError(fmt.Sprintf("%s must be unique", field), map[string]any{"field": field})
		}
		seen[id] = true
		unique = append(unique, id)
	}
	return unique, nil
}

func normalizeLifestyleAnswer(question LifestyleQuestion, raw json.RawMessage) (datatypes.JSON, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, validationError("Lifestyle answer cannot be empty", map[string]any{"field": "answer"})
	}

	options, err := lifestyleOptions(question.Options)
	if err != nil {
		return nil, err
	}
	optionSet := make(map[string]bool, len(options))
	for _, option := range options {
		optionSet[option] = true
	}

	switch question.AnswerType {
	case AnswerTypeSingleChoice:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, validationError("Lifestyle answer must be a string", map[string]any{"field": "answer"})
		}
		value = strings.TrimSpace(value)
		if value == "" || !optionSet[value] {
			return nil, validationError("Lifestyle answer option is invalid", map[string]any{"field": "answer"})
		}
		body, _ := json.Marshal(value)
		return datatypes.JSON(body), nil
	case AnswerTypeMultipleChoice:
		var values []string
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, validationError("Lifestyle answer must be an array", map[string]any{"field": "answer"})
		}
		if len(values) == 0 {
			return nil, validationError("Lifestyle answer cannot be empty", map[string]any{"field": "answer"})
		}
		seen := make(map[string]bool, len(values))
		for i, value := range values {
			value = strings.TrimSpace(value)
			if value == "" || !optionSet[value] {
				return nil, validationError("Lifestyle answer option is invalid", map[string]any{"field": "answer"})
			}
			if seen[value] {
				return nil, validationError("Lifestyle answer values must be unique", map[string]any{"field": "answer"})
			}
			seen[value] = true
			values[i] = value
		}
		body, _ := json.Marshal(values)
		return datatypes.JSON(body), nil
	case AnswerTypeText:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, validationError("Lifestyle answer must be text", map[string]any{"field": "answer"})
		}
		value = strings.TrimSpace(value)
		if value == "" || len(value) > 300 {
			return nil, validationError("Lifestyle text answer must be 1 to 300 characters", map[string]any{"field": "answer"})
		}
		body, _ := json.Marshal(value)
		return datatypes.JSON(body), nil
	default:
		return nil, validationError("Lifestyle question answer type is invalid", map[string]any{"field": "answerType"})
	}
}

func ParseDateOfBirth(raw string) (time.Time, error) {
	parsed, err := time.ParseInLocation(dateLayout, strings.TrimSpace(raw), time.UTC)
	if err != nil {
		return time.Time{}, err
	}
	return dateOnly(parsed), nil
}

func IsAdultDOB(dob time.Time, now time.Time) bool {
	minimumDOB := dateOnly(now.UTC()).AddDate(-18, 0, 0)
	return !dateOnly(dob).After(minimumDOB)
}

func dateOnly(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func validGender(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case GenderMale, GenderFemale, GenderNonBinary, GenderOther:
		return true
	default:
		return false
	}
}

func validLookingForGender(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case LookingForMale, LookingForFemale, LookingForEveryone:
		return true
	default:
		return false
	}
}

func validRelationshipGoal(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case RelationshipSerious, RelationshipLongTerm, RelationshipFriendship, RelationshipCasual, RelationshipNotSure:
		return true
	default:
		return false
	}
}

func optionalString(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func toProfileResponse(profile Profile, completion CompletionResponse) ProfileResponse {
	var dob *string
	if profile.DateOfBirth != nil {
		formatted := dateOnly(*profile.DateOfBirth).Format(dateLayout)
		dob = &formatted
	}

	percentage := profile.CompletionPercentage
	if completion.Percentage > 0 || len(completion.Missing) > 0 {
		percentage = completion.Percentage
	}

	return ProfileResponse{
		UUID:                 profile.UUID.String(),
		DisplayName:          profile.DisplayName,
		DateOfBirth:          dob,
		Gender:               profile.Gender,
		LookingForGender:     profile.LookingForGender,
		Bio:                  profile.Bio,
		HeightCM:             profile.HeightCM,
		Education:            profile.Education,
		JobTitle:             profile.JobTitle,
		Company:              profile.Company,
		City:                 profile.City,
		Country:              profile.Country,
		RelationshipGoal:     profile.RelationshipGoal,
		ShowAge:              profile.ShowAge,
		ShowDistance:         profile.ShowDistance,
		CompletionPercentage: percentage,
		ProfileStatus:        profile.ProfileStatus,
		DiscoveryEligible:    profile.DiscoveryEligible,
	}
}

func toInterestResponse(interest Interest) InterestResponse {
	return InterestResponse{
		ID:          interest.ID,
		UUID:        interest.UUID.String(),
		InterestKey: interest.InterestKey,
		Name:        interest.Name,
		Icon:        interest.Icon,
		CategoryID:  interest.CategoryID,
		SortOrder:   interest.SortOrder,
	}
}

func toPromptQuestionResponse(question ProfilePromptQuestion) PromptQuestionResponse {
	return PromptQuestionResponse{
		ID:        question.ID,
		UUID:      question.UUID.String(),
		PromptKey: question.PromptKey,
		Question:  question.Question,
		SortOrder: question.SortOrder,
	}
}

func toLifestyleQuestionResponse(question LifestyleQuestion) (LifestyleQuestionResponse, error) {
	options, err := lifestyleOptions(question.Options)
	if err != nil {
		return LifestyleQuestionResponse{}, err
	}
	return LifestyleQuestionResponse{
		ID:          question.ID,
		UUID:        question.UUID.String(),
		QuestionKey: question.QuestionKey,
		Question:    question.Question,
		AnswerType:  question.AnswerType,
		Options:     options,
		SortOrder:   question.SortOrder,
	}, nil
}

func lifestyleOptions(raw datatypes.JSON) ([]string, error) {
	var options []string
	if len(raw) == 0 {
		return options, nil
	}
	if err := json.Unmarshal(raw, &options); err != nil {
		return nil, err
	}
	return options, nil
}

func unmarshalJSONAny(raw datatypes.JSON) (any, error) {
	var value any
	if len(raw) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func AsServiceError(err error) (*ServiceError, bool) {
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr, true
	}
	return nil, false
}
