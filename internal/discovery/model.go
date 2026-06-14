package discovery

const (
	GenderMale      = "MALE"
	GenderFemale    = "FEMALE"
	GenderNonBinary = "NON_BINARY"
	GenderOther     = "OTHER"
	GenderEveryone  = "EVERYONE"

	UserStatusActive    = "ACTIVE"
	UserStatusSuspended = "SUSPENDED"
	UserStatusBanned    = "BANNED"
	UserStatusDeleted   = "DELETED"

	ProfileStatusDraft       = "DRAFT"
	ProfileStatusActive      = "ACTIVE"
	ProfileStatusPaused      = "PAUSED"
	ProfileStatusUnderReview = "UNDER_REVIEW"
	ProfileStatusRejected    = "REJECTED"
)

type DiscoverabilityMode string

const (
	DiscoverabilityModeFeed          DiscoverabilityMode = "FEED"
	DiscoverabilityModeAction        DiscoverabilityMode = "ACTION"
	DiscoverabilityModeProfileDetail DiscoverabilityMode = "PROFILE_DETAIL"
)
