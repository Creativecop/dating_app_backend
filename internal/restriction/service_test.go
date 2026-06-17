package restriction

import "testing"

func TestRestrictionBlocksAction(t *testing.T) {
	tests := []struct {
		name            string
		restrictionType string
		action          string
		blocked         bool
	}{
		{"full blocks login", TypeFullPlatformBan, ActionLogin, true},
		{"full blocks authenticated request", TypeFullPlatformBan, ActionAuthenticated, true},
		{"full blocks socket", TypeFullPlatformBan, ActionSocketConnect, true},
		{"comment blocks comments", TypeCommentBan, ActionSendComment, true},
		{"comment does not block auth", TypeCommentBan, ActionAuthenticated, false},
		{"comment does not block socket", TypeCommentBan, ActionSocketConnect, false},
		{"gift blocks gift", TypeGiftSendBan, ActionSendGift, true},
		{"gift does not block agency", TypeGiftSendBan, ActionAgencyManage, false},
		{"agency blocks agency", TypeAgencyOperationBan, ActionAgencyManage, true},
		{"reseller blocks topup", TypeResellerOperationBan, ActionResellerTopup, true},
		{"live blocks create", TypeLiveCreateBan, ActionCreateLive, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := RestrictionBlocksAction(test.restrictionType, test.action); got != test.blocked {
				t.Fatalf("blocked = %v, want %v", got, test.blocked)
			}
		})
	}
}

func TestRestrictionValidation(t *testing.T) {
	if !IsValidRestrictionType(TypeFullPlatformBan) {
		t.Fatal("FULL_PLATFORM_BAN should be valid")
	}
	if IsValidRestrictionType("CHAT_MESSAGE_BAN") {
		t.Fatal("CHAT_MESSAGE_BAN should not be part of Phase 13B")
	}
	if !IsValidStatus(StatusExpired) {
		t.Fatal("EXPIRED status should be valid")
	}
}
