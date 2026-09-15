package ecs

import (
	"time"
)

type Expiry struct {
	Remaining time.Duration
}

func (e Expiry) Name() ComponentName {
	return "Expiry"
}

type ExpirySystem struct {
}

// NewExpirySystem constructs a new expiry system.
func NewExpirySystem() *ExpirySystem {
	return &ExpirySystem{}
}

// Update the ExpirySystem.
func (es *ExpirySystem) Update(w *World, elapsed time.Duration) error {
	for _, e := range w.Get(Expiry{}.Name()) {
		expiry := Get[Expiry](w, e)

		expiry.Remaining -= elapsed
		if expiry.Remaining <= 0 {
			w.DestroyEntity(e)
		}
	}
	return nil
}
