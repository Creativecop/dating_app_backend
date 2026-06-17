package admin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neoscoder/aura-backend/internal/config"
)

func TestCapabilitiesReflectConfiguredModules(t *testing.T) {
	service := NewService(nil, config.JWTConfig{})
	service.SetModuleCapabilities(AdminModuleCapabilities{
		TrustSafety:                  true,
		SubscriptionPayments:         true,
		SubscriptionPaymentAnalytics: true,
	})

	result, err := service.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities returned error: %v", err)
	}
	if !result.Modules.TrustSafety {
		t.Fatal("trustSafety should be enabled")
	}
	if !result.Modules.SubscriptionPayments || !result.Modules.SubscriptionPaymentAnalytics {
		t.Fatal("subscription payment capabilities should be enabled")
	}
	if result.Modules.Games || result.Modules.PKBattle || result.Modules.GreedyGame {
		t.Fatalf("game capabilities should be disabled in this repo: %#v", result.Modules)
	}
	if result.Modules.Wallet || result.Modules.Gift || result.Modules.Agency || result.Modules.Reseller || result.Modules.Live || result.Modules.LiveComments || result.Modules.ChatModeration {
		t.Fatalf("unexpected enabled module: %#v", result.Modules)
	}
}

func TestGamePermissionsDoNotEnableGameCapabilities(t *testing.T) {
	if !RoleHasPermission(RoleSupportAgent, PermissionGamesRead) {
		t.Fatal("support agent should have future games.read permission")
	}

	service := NewService(nil, config.JWTConfig{})
	result, err := service.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities returned error: %v", err)
	}
	if result.Modules.Games || result.Modules.PKBattle || result.Modules.GreedyGame {
		t.Fatalf("permissions must not imply game capabilities: %#v", result.Modules)
	}
}

func TestPKRuntimeArtifactsAreNotPresent(t *testing.T) {
	if _, err := os.Stat("../pk"); !os.IsNotExist(err) {
		t.Fatalf("internal/pk must not exist in this repo; err=%v", err)
	}

	matches, err := filepath.Glob("../../migrations/*pk*")
	if err != nil {
		t.Fatalf("glob pk migrations: %v", err)
	}
	if len(matches) > 0 {
		t.Fatalf("PK migrations must not be introduced in this repo: %#v", matches)
	}
}

func TestPKRoutesAreNotRegistered(t *testing.T) {
	routerSource, err := os.ReadFile("../router/router.go")
	if err != nil {
		t.Fatalf("read router source: %v", err)
	}
	if strings.Contains(string(routerSource), "/pk") || strings.Contains(string(routerSource), "pk-battles") {
		t.Fatal("PK routes must not be registered in this repo")
	}
}
