package profile

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	GenderMale      = "MALE"
	GenderFemale    = "FEMALE"
	GenderNonBinary = "NON_BINARY"
	GenderOther     = "OTHER"

	LookingForMale     = "MALE"
	LookingForFemale   = "FEMALE"
	LookingForEveryone = "EVERYONE"

	RelationshipSerious    = "SERIOUS_RELATIONSHIP"
	RelationshipLongTerm   = "LONG_TERM"
	RelationshipFriendship = "FRIENDSHIP"
	RelationshipCasual     = "CASUAL"
	RelationshipNotSure    = "NOT_SURE"

	ProfileStatusDraft       = "DRAFT"
	ProfileStatusActive      = "ACTIVE"
	ProfileStatusPaused      = "PAUSED"
	ProfileStatusUnderReview = "UNDER_REVIEW"
	ProfileStatusRejected    = "REJECTED"

	AnswerTypeSingleChoice   = "SINGLE_CHOICE"
	AnswerTypeMultipleChoice = "MULTIPLE_CHOICE"
	AnswerTypeText           = "TEXT"
)

type Profile struct {
	ID                            uint64    `gorm:"primaryKey"`
	UUID                          uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID                        uint64    `gorm:"uniqueIndex"`
	DisplayName                   *string
	DateOfBirth                   *time.Time `gorm:"type:date"`
	Gender                        *string
	LookingForGender              *string
	Bio                           *string
	HeightCM                      *int
	Education                     *string
	JobTitle                      *string
	Company                       *string
	City                          *string
	Country                       *string
	RelationshipGoal              *string
	ShowAge                       bool
	ShowDistance                  bool
	CompletionPercentage          int
	ProfileStatus                 string
	DiscoveryEligible             bool
	DiscoveryEligibilityUpdatedAt *time.Time
	CompletedAt                   *time.Time
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}

func (Profile) TableName() string {
	return "profiles"
}

type InterestCategory struct {
	ID          uint64    `gorm:"primaryKey"`
	UUID        uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	CategoryKey string
	Name        string
	SortOrder   int
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Interests   []Interest `gorm:"foreignKey:CategoryID"`
}

func (InterestCategory) TableName() string {
	return "interest_categories"
}

type Interest struct {
	ID          uint64    `gorm:"primaryKey"`
	UUID        uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	CategoryID  *uint64
	Category    *InterestCategory
	InterestKey string
	Name        string
	Icon        *string
	SortOrder   int
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Interest) TableName() string {
	return "interests"
}

type UserInterest struct {
	ID         uint64    `gorm:"primaryKey"`
	UUID       uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID     uint64
	InterestID uint64
	Interest   Interest
	CreatedAt  time.Time
}

func (UserInterest) TableName() string {
	return "user_interests"
}

type ProfilePromptQuestion struct {
	ID        uint64    `gorm:"primaryKey"`
	UUID      uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	PromptKey string
	Question  string
	SortOrder int
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ProfilePromptQuestion) TableName() string {
	return "profile_prompt_questions"
}

type UserProfilePrompt struct {
	ID               uint64    `gorm:"primaryKey"`
	UUID             uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID           uint64
	PromptQuestionID uint64
	Question         ProfilePromptQuestion `gorm:"foreignKey:PromptQuestionID"`
	Answer           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (UserProfilePrompt) TableName() string {
	return "user_profile_prompts"
}

type LifestyleQuestion struct {
	ID          uint64    `gorm:"primaryKey"`
	UUID        uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	QuestionKey string
	Question    string
	AnswerType  string
	Options     datatypes.JSON `gorm:"type:jsonb"`
	SortOrder   int
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (LifestyleQuestion) TableName() string {
	return "lifestyle_questions"
}

type UserLifestyleAnswer struct {
	ID                  uint64    `gorm:"primaryKey"`
	UUID                uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID              uint64
	LifestyleQuestionID uint64
	Question            LifestyleQuestion `gorm:"foreignKey:LifestyleQuestionID"`
	Answer              datatypes.JSON    `gorm:"type:jsonb"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (UserLifestyleAnswer) TableName() string {
	return "user_lifestyle_answers"
}
