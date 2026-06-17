package admin

import (
	"context"
	"testing"

	"github.com/neoscoder/aura-backend/internal/config"
)

func TestCapabilitiesReflectConfiguredModules(t *testing.T) {
	service := NewService(nil, config.JWTConfig{})
	service.SetModuleCapabilities(AdminModuleCapabilities{TrustSafety: true})

	result, err := service.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities returned error: %v", err)
	}
	if !result.Modules.TrustSafety {
		t.Fatal("trustSafety should be enabled")
	}
	if result.Modules.Wallet || result.Modules.Gift || result.Modules.Agency || result.Modules.Reseller || result.Modules.Live || result.Modules.LiveComments || result.Modules.ChatModeration {
		t.Fatalf("unexpected enabled module: %#v", result.Modules)
	}
}
