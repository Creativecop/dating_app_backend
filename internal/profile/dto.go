package profile

import (
	"encoding/json"
)

type PatchProfileRequest struct {
	DisplayName      *string `json:"displayName"`
	DateOfBirth      *string `json:"dateOfBirth"`
	Gender           *string `json:"gender"`
	LookingForGender *string `json:"lookingForGender"`
	Bio              *string `json:"bio"`
	HeightCM         *int    `json:"heightCm"`
	Education        *string `json:"education"`
	JobTitle         *string `json:"jobTitle"`
	Company          *string `json:"company"`
	City             *string `json:"city"`
	Country          *string `json:"country"`
	RelationshipGoal *string `json:"relationshipGoal"`
	ShowAge          *bool   `json:"showAge"`
	ShowDistance     *bool   `json:"showDistance"`

	present map[string]bool
	nulls   map[string]bool
}

func (r *PatchProfileRequest) UnmarshalJSON(data []byte) error {
	type alias PatchProfileRequest
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*r = PatchProfileRequest(aux)
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

func (r PatchProfileRequest) Has(field string) bool {
	return r.present[field]
}

func (r PatchProfileRequest) IsNull(field string) bool {
	return r.nulls[field]
}

type UpdateInterestsRequest struct {
	InterestIDs []uint64 `json:"interestIds" binding:"required"`
}

type UpdatePromptsRequest struct {
	Prompts []PromptAnswerRequest `json:"prompts" binding:"required"`
}

type PromptAnswerRequest struct {
	PromptQuestionID uint64 `json:"promptQuestionId" binding:"required"`
	Answer           string `json:"answer" binding:"required"`
}

type UpdateLifestyleRequest struct {
	Answers []LifestyleAnswerRequest `json:"answers"`
}

type LifestyleAnswerRequest struct {
	LifestyleQuestionID uint64          `json:"lifestyleQuestionId" binding:"required"`
	Answer              json.RawMessage `json:"answer" binding:"required"`
}

type ProfileMeResponse struct {
	Profile          ProfileResponse           `json:"profile"`
	Interests        []InterestResponse        `json:"interests"`
	Prompts          []UserPromptResponse      `json:"prompts"`
	LifestyleAnswers []LifestyleAnswerResponse `json:"lifestyleAnswers"`
	Completion       CompletionResponse        `json:"completion"`
}

type ProfileResponse struct {
	UUID                 string  `json:"uuid"`
	DisplayName          *string `json:"displayName"`
	DateOfBirth          *string `json:"dateOfBirth"`
	Gender               *string `json:"gender"`
	LookingForGender     *string `json:"lookingForGender"`
	Bio                  *string `json:"bio"`
	HeightCM             *int    `json:"heightCm"`
	Education            *string `json:"education"`
	JobTitle             *string `json:"jobTitle"`
	Company              *string `json:"company"`
	City                 *string `json:"city"`
	Country              *string `json:"country"`
	RelationshipGoal     *string `json:"relationshipGoal"`
	ShowAge              bool    `json:"showAge"`
	ShowDistance         bool    `json:"showDistance"`
	CompletionPercentage int     `json:"completionPercentage"`
	ProfileStatus        string  `json:"profileStatus"`
	DiscoveryEligible    bool    `json:"discoveryEligible"`
}

type InterestCatalogResponse struct {
	Categories []InterestCategoryResponse `json:"categories"`
}

type InterestCategoryResponse struct {
	ID          uint64             `json:"id"`
	UUID        string             `json:"uuid"`
	CategoryKey string             `json:"categoryKey"`
	Name        string             `json:"name"`
	SortOrder   int                `json:"sortOrder"`
	Interests   []InterestResponse `json:"interests"`
}

type InterestResponse struct {
	ID          uint64  `json:"id"`
	UUID        string  `json:"uuid"`
	InterestKey string  `json:"interestKey"`
	Name        string  `json:"name"`
	Icon        *string `json:"icon"`
	CategoryID  *uint64 `json:"categoryId"`
	SortOrder   int     `json:"sortOrder"`
}

type PromptQuestionResponse struct {
	ID        uint64 `json:"id"`
	UUID      string `json:"uuid"`
	PromptKey string `json:"promptKey"`
	Question  string `json:"question"`
	SortOrder int    `json:"sortOrder"`
}

type UserPromptResponse struct {
	PromptQuestion PromptQuestionResponse `json:"promptQuestion"`
	Answer         string                 `json:"answer"`
}

type LifestyleQuestionResponse struct {
	ID          uint64   `json:"id"`
	UUID        string   `json:"uuid"`
	QuestionKey string   `json:"questionKey"`
	Question    string   `json:"question"`
	AnswerType  string   `json:"answerType"`
	Options     []string `json:"options"`
	SortOrder   int      `json:"sortOrder"`
}

type LifestyleAnswerResponse struct {
	LifestyleQuestion LifestyleQuestionResponse `json:"lifestyleQuestion"`
	Answer            any                       `json:"answer"`
}

type CompletionResponse struct {
	Percentage int      `json:"percentage"`
	Status     string   `json:"status"`
	Missing    []string `json:"missing"`
}
