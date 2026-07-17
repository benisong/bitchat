package contribution

import (
	"errors"
	"math"
	"testing"
)

func TestPermanentContributionLifecycle(t *testing.T) {
	state := NewState("owner")
	if err := state.Earn(10); err != nil {
		t.Fatal(err)
	}
	if err := state.Spend(3); err != nil {
		t.Fatal(err)
	}
	if err := state.Burn(2); err != nil {
		t.Fatal(err)
	}
	snapshot := state.Snapshot()
	if snapshot.SpendableCredit != 5 || snapshot.LifetimeContribution != 10 || snapshot.BurnedCredit != 2 {
		t.Fatalf("unexpected credit snapshot: %+v", snapshot)
	}
	if snapshot.Version != 3 {
		t.Fatalf("version = %d, want 3", snapshot.Version)
	}
}

func TestContributionRejectsNegativeAndInsufficientAmounts(t *testing.T) {
	state := NewState("owner")
	if err := state.Earn(-1); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("negative earn returned %v", err)
	}
	if err := state.Spend(-1); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("negative spend returned %v", err)
	}
	if err := state.Burn(-1); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("negative burn returned %v", err)
	}
	if state.CanSpend(-1) {
		t.Fatal("negative amount was spendable")
	}
	if err := state.Spend(1); !errors.Is(err, ErrInsufficientCredit) {
		t.Fatalf("overspend returned %v", err)
	}
}

func TestContributionDetectsOverflow(t *testing.T) {
	state := NewState("owner")
	state.spendableCredit = math.MaxInt64
	state.lifetimeContribution = math.MaxInt64
	if err := state.Earn(1); !errors.Is(err, ErrCreditStateOverflow) {
		t.Fatalf("overflow returned %v", err)
	}
}
