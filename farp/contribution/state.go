package contribution

import (
	"errors"
	"math"
	"sync"
	"time"
)

var (
	ErrInvalidAmount       = errors.New("credit amount must be positive")
	ErrInsufficientCredit  = errors.New("insufficient spendable credit")
	ErrCreditStateOverflow = errors.New("credit state overflow")
)

type Snapshot struct {
	Owner                string    `json:"owner"`
	Version              uint64    `json:"version"`
	SpendableCredit      int64     `json:"spendable_credit"`
	LifetimeContribution int64     `json:"lifetime_contribution"`
	BurnedCredit         int64     `json:"burned_credit"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// State is the local projection of permanent, non-transferable contribution credit.
type State struct {
	mu sync.RWMutex

	owner                string
	version              uint64
	spendableCredit      int64
	lifetimeContribution int64
	burnedCredit         int64
	updatedAt            time.Time
}

func NewState(owner string) *State {
	return &State{owner: owner, updatedAt: time.Now().UTC()}
}

func (s *State) Earn(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.spendableCredit > math.MaxInt64-amount || s.lifetimeContribution > math.MaxInt64-amount {
		return ErrCreditStateOverflow
	}
	s.spendableCredit += amount
	s.lifetimeContribution += amount
	s.advanceVersion()
	return nil
}

func (s *State) CanSpend(amount int64) bool {
	if amount <= 0 {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.spendableCredit >= amount
}

func (s *State) Spend(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.spendableCredit < amount {
		return ErrInsufficientCredit
	}
	s.spendableCredit -= amount
	s.advanceVersion()
	return nil
}

func (s *State) Burn(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.spendableCredit < amount {
		return ErrInsufficientCredit
	}
	if s.burnedCredit > math.MaxInt64-amount {
		return ErrCreditStateOverflow
	}
	s.spendableCredit -= amount
	s.burnedCredit += amount
	s.advanceVersion()
	return nil
}

func (s *State) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Snapshot{
		Owner:                s.owner,
		Version:              s.version,
		SpendableCredit:      s.spendableCredit,
		LifetimeContribution: s.lifetimeContribution,
		BurnedCredit:         s.burnedCredit,
		UpdatedAt:            s.updatedAt,
	}
}

func (s *State) advanceVersion() {
	s.version++
	s.updatedAt = time.Now().UTC()
}
