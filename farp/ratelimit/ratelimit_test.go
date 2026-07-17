package ratelimit

import (
	"testing"

	"github.com/benisong/bitchat/internal/config"
)

func TestUnifiedQuotaDefaultsToHardMinimum(t *testing.T) {
	manager := NewManager()
	for i := 0; i < config.MinQuotaPerWindow; i++ {
		if !manager.Deduce("peer", "operation") {
			t.Fatalf("request %d unexpectedly rejected", i+1)
		}
	}
	if manager.Deduce("peer", "different-operation") {
		t.Fatal("different operation bypassed the unified quota")
	}
}

func TestCapacityCannotEscapeHardGate(t *testing.T) {
	manager := NewManager()
	manager.SetCapacity(100)
	_, capacity := manager.Peek("upper-bound-peer")
	if capacity != config.MaxQuotaPerWindow {
		t.Fatalf("upper capacity = %.0f, want %d", capacity, config.MaxQuotaPerWindow)
	}

	manager.SetCapacity(0)
	_, capacity = manager.Peek("upper-bound-peer")
	if capacity != config.MinQuotaPerWindow {
		t.Fatalf("existing bucket capacity = %.0f, want %d", capacity, config.MinQuotaPerWindow)
	}
	_, capacity = manager.Peek("lower-bound-peer")
	if capacity != config.MinQuotaPerWindow {
		t.Fatalf("new bucket capacity = %.0f, want %d", capacity, config.MinQuotaPerWindow)
	}
}
