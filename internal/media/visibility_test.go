package media

import (
	"context"
	"testing"
)

type fakeMediaAuthorizer struct {
	allowed bool
}

func (f fakeMediaAuthorizer) CanViewProfileMedia(context.Context, uint64, uint64, string, string) bool {
	return f.allowed
}

func TestCanViewProfileMediaAllowsAnyRegisteredAuthorizer(t *testing.T) {
	service := &Service{}
	if service.canViewProfileMedia(context.Background(), 1, 2, "media", VariantDisplay) {
		t.Fatal("expected no authorizers to deny access")
	}

	service.AddMediaVisibilityAuthorizer(fakeMediaAuthorizer{allowed: false})
	service.AddMediaVisibilityAuthorizer(fakeMediaAuthorizer{allowed: true})

	if !service.canViewProfileMedia(context.Background(), 1, 2, "media", VariantDisplay) {
		t.Fatal("expected any registered authorizer to allow access")
	}
}
