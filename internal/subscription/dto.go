package subscription

import "time"

type PlanResponse struct {
	PlanUUID     string       `json:"planUuid"`
	PlanCode     string       `json:"planCode"`
	Name         string       `json:"name"`
	Description  *string      `json:"description"`
	PriceAmount  int          `json:"priceAmount"`
	Currency     string       `json:"currency"`
	DurationDays int          `json:"durationDays"`
	Entitlements Entitlements `json:"entitlements"`
}

type PlanListResponse struct {
	Items []PlanResponse `json:"items"`
}

type CreateManualPaymentRequest struct {
	PlanCode         string  `json:"planCode" binding:"required"`
	PaymentProvider  string  `json:"paymentProvider" binding:"required"`
	PaymentReference string  `json:"paymentReference" binding:"required"`
	PayerPhone       *string `json:"payerPhone"`
	Note             *string `json:"note"`
}

type PaymentRequestResponse struct {
	PaymentRequestUUID string     `json:"paymentRequestUuid"`
	PlanCode           string     `json:"planCode"`
	PlanName           string     `json:"planName"`
	PriceAmount        int        `json:"priceAmount"`
	Currency           string     `json:"currency"`
	DurationDays       int        `json:"durationDays"`
	PaymentProvider    string     `json:"paymentProvider"`
	PaymentReference   *string    `json:"paymentReference"`
	PayerPhone         *string    `json:"payerPhone,omitempty"`
	Note               *string    `json:"note,omitempty"`
	Status             string     `json:"status"`
	SubmittedAt        time.Time  `json:"submittedAt"`
	ReviewedAt         *time.Time `json:"reviewedAt,omitempty"`
	RejectionReason    *string    `json:"rejectionReason,omitempty"`
	SubscriptionUUID   *string    `json:"subscriptionUuid,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
}

type PaymentRequestListResponse struct {
	Items []PaymentRequestResponse `json:"items"`
}

type SubscriptionResponse struct {
	SubscriptionUUID string       `json:"subscriptionUuid"`
	PlanCode         string       `json:"planCode"`
	PlanName         string       `json:"planName"`
	Status           string       `json:"status"`
	StartsAt         time.Time    `json:"startsAt"`
	ExpiresAt        time.Time    `json:"expiresAt"`
	Entitlements     Entitlements `json:"entitlements"`
}

type CurrentSubscriptionResponse struct {
	Subscription *SubscriptionResponse `json:"subscription"`
}

type EntitlementsResponse struct {
	IsPremium    bool         `json:"isPremium"`
	PlanCode     *string      `json:"planCode"`
	ExpiresAt    *time.Time   `json:"expiresAt"`
	Entitlements Entitlements `json:"entitlements"`
}

type UsageResponse struct {
	Date      string                 `json:"date"`
	Usage     UsageUsedResponse      `json:"usage"`
	Remaining UsageRemainingResponse `json:"remaining"`
}

type UsageUsedResponse struct {
	LikesUsedToday            int `json:"likesUsedToday"`
	SuperLikesUsedToday       int `json:"superLikesUsedToday"`
	AudioCallSecondsUsedToday int `json:"audioCallSecondsUsedToday"`
	VideoCallSecondsUsedToday int `json:"videoCallSecondsUsedToday"`
}

type UsageRemainingResponse struct {
	LikesRemainingToday            int `json:"likesRemainingToday"`
	SuperLikesRemainingToday       int `json:"superLikesRemainingToday"`
	AudioCallSecondsRemainingToday int `json:"audioCallSecondsRemainingToday"`
	VideoCallSecondsRemainingToday int `json:"videoCallSecondsRemainingToday"`
}

type PremiumStatusResponse struct {
	IsPremium bool       `json:"isPremium"`
	PlanCode  *string    `json:"planCode"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type AdminPaymentRequestListResponse struct {
	Items []AdminPaymentRequestResponse `json:"items"`
}

type AdminPaymentRequestResponse struct {
	PaymentRequestUUID string       `json:"paymentRequestUuid"`
	UserUUID           string       `json:"userUuid"`
	PlanCode           string       `json:"planCode"`
	PlanName           string       `json:"planName"`
	PriceAmount        int          `json:"priceAmount"`
	Currency           string       `json:"currency"`
	DurationDays       int          `json:"durationDays"`
	Entitlements       Entitlements `json:"entitlements"`
	PaymentProvider    string       `json:"paymentProvider"`
	PaymentReference   *string      `json:"paymentReference"`
	PayerPhone         *string      `json:"payerPhone,omitempty"`
	Note               *string      `json:"note,omitempty"`
	Status             string       `json:"status"`
	SubmittedAt        time.Time    `json:"submittedAt"`
	ReviewedAt         *time.Time   `json:"reviewedAt,omitempty"`
	RejectionReason    *string      `json:"rejectionReason,omitempty"`
	SubscriptionUUID   *string      `json:"subscriptionUuid,omitempty"`
	CreatedAt          time.Time    `json:"createdAt"`
}

type ReviewPaymentRequest struct {
	Note string `json:"note"`
}

type RejectPaymentRequest struct {
	Reason string `json:"reason" binding:"required"`
}
