package profile

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithDB(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) EnsureProfile(ctx context.Context, userID uint64) error {
	now := time.Now().UTC()
	profile := Profile{
		UUID:                 uuid.New(),
		UserID:               userID,
		ShowAge:              true,
		ShowDistance:         true,
		CompletionPercentage: 0,
		ProfileStatus:        ProfileStatusDraft,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&profile).Error
}

func (r *Repository) GetProfile(ctx context.Context, userID uint64) (*Profile, error) {
	var item Profile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) ListInterestCategories(ctx context.Context) ([]InterestCategory, error) {
	var categories []InterestCategory
	err := r.db.WithContext(ctx).
		Preload("Interests", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).Order("sort_order ASC").Order("name ASC")
		}).
		Where("is_active = ?", true).
		Order("sort_order ASC").
		Order("name ASC").
		Find(&categories).Error
	return categories, err
}

func (r *Repository) ListPromptQuestions(ctx context.Context) ([]ProfilePromptQuestion, error) {
	var questions []ProfilePromptQuestion
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("sort_order ASC").
		Order("id ASC").
		Find(&questions).Error
	return questions, err
}

func (r *Repository) ListLifestyleQuestions(ctx context.Context) ([]LifestyleQuestion, error) {
	var questions []LifestyleQuestion
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("sort_order ASC").
		Order("id ASC").
		Find(&questions).Error
	return questions, err
}

func (r *Repository) UserInterests(ctx context.Context, userID uint64) ([]UserInterest, error) {
	var rows []UserInterest
	err := r.db.WithContext(ctx).
		Preload("Interest").
		Joins("JOIN interests ON interests.id = user_interests.interest_id AND interests.is_active = TRUE").
		Where("user_interests.user_id = ?", userID).
		Order("interests.sort_order ASC").
		Find(&rows).Error
	return rows, err
}

func (r *Repository) UserPrompts(ctx context.Context, userID uint64) ([]UserProfilePrompt, error) {
	var rows []UserProfilePrompt
	err := r.db.WithContext(ctx).
		Preload("Question").
		Joins("JOIN profile_prompt_questions ON profile_prompt_questions.id = user_profile_prompts.prompt_question_id AND profile_prompt_questions.is_active = TRUE").
		Where("user_profile_prompts.user_id = ?", userID).
		Order("profile_prompt_questions.sort_order ASC").
		Find(&rows).Error
	return rows, err
}

func (r *Repository) UserLifestyleAnswers(ctx context.Context, userID uint64) ([]UserLifestyleAnswer, error) {
	var rows []UserLifestyleAnswer
	err := r.db.WithContext(ctx).
		Preload("Question").
		Joins("JOIN lifestyle_questions ON lifestyle_questions.id = user_lifestyle_answers.lifestyle_question_id AND lifestyle_questions.is_active = TRUE").
		Where("user_lifestyle_answers.user_id = ?", userID).
		Order("lifestyle_questions.sort_order ASC").
		Find(&rows).Error
	return rows, err
}

func (r *Repository) CountActiveInterests(ctx context.Context, tx *gorm.DB, userID uint64) (int64, error) {
	var count int64
	err := tx.WithContext(ctx).
		Table("user_interests").
		Joins("JOIN interests ON interests.id = user_interests.interest_id AND interests.is_active = TRUE").
		Where("user_interests.user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *Repository) CountActivePrompts(ctx context.Context, tx *gorm.DB, userID uint64) (int64, error) {
	var count int64
	err := tx.WithContext(ctx).
		Table("user_profile_prompts").
		Joins("JOIN profile_prompt_questions ON profile_prompt_questions.id = user_profile_prompts.prompt_question_id AND profile_prompt_questions.is_active = TRUE").
		Where("user_profile_prompts.user_id = ?", userID).
		Count(&count).Error
	return count, err
}
