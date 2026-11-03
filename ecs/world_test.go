package ecs

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestSystems(t *testing.T) {

	t.Run("Register", func(t *testing.T) {
		w := NewWorld()
		if err := w.Register(NewExpirySystem(), NewChildrenSystem()); err != nil {
			t.Fatalf("want no errors from Register, got %v", err)
		}
		t.Skip()

		err := w.Register(NewExpirySystem(), NewChildrenSystem())
		want := `duplicate system registered "ecs.ExpirySystem"
duplicate system registered "ecs.ChildrenSystem"
`

		if err != errors.New(want) {
			t.Errorf("\nwant %q\ngot  %q", want, err)
		}
	})

	t.Run("Update", func(t *testing.T) {

		w := NewWorld()
		w.Register(NewExpirySystem(), NewChildrenSystem())

		e := w.CreateEntity()
		kids := Children{}
		for i := 0; i < 3; i++ {
			kids.Value = append(kids.Value, w.CreateEntity())
		}
		Add(w, e, &kids)
		Add(w, e, &Expiry{5 * time.Second})
		// The world now has 4 Entites, and
		// the primary Entity now has two Components

		w.Update(4 * time.Second)

		if w.Len() != 4 {
			t.Fatal("incorrect number of entities in test setup")
		}

		w.Update(5*time.Second + 1*time.Millisecond)

		if w.Len() != 3 {
			t.Fatalf("want no entities, got %d", w.Len())
		}
	})

	t.Run("Store and retrieve components", func(t *testing.T) {
		w := NewWorld()
		e := w.CreateEntity()
		Add(w, e, &Expiry{5 * time.Second})

		expiry1 := Get[Expiry](w, e)
		expiry1.Remaining = 4 * time.Second

		expiry2 := Get[Expiry](w, e)
		if fmt.Sprintf("%v", expiry2) != fmt.Sprintf("%v", expiry1) {
			t.Errorf("pointer mismatch! want %p, got %p", expiry1, expiry2)
		}
		if expiry2.Remaining != 4*time.Second {
			t.Errorf("time mismatch! want %v, got %v", 4*time.Second, expiry2.Remaining)
		}
	})

	t.Run("Destroy dependants", func(t *testing.T) {
		w := NewWorld()
		e := w.CreateEntity()
		for i := 0; i < 3; i++ {
			ce := w.CreateEntity()
			w.Dependency(e, ce)
		}

		if w.Len() != 4 {
			t.Errorf("wrong test setup of entity length, want %d, got %d", 4, w.Len())
		}

		w.DestroyEntity(e)

		if w.Len() != 0 {
			t.Errorf("wrong entity length after Destroy, want %d, got %d", 0, w.Len())
		}
	})

	t.Run("System execution order", func(t *testing.T) {
		w := NewWorld()

		order := []string{}
		s1 := &mockSys1{onUpdate: func() { order = append(order, "Steering") }}
		s2 := &mockSys2{onUpdate: func() { order = append(order, "Motion") }}
		s3 := &mockSys3{onUpdate: func() { order = append(order, "Collision") }}
		s4 := &mockSys4{onUpdate: func() { order = append(order, "Reversion") }}

		w.Register(s1, s2, s3, s4)

		for run := 0; run < 10; run++ {
			order = order[:0]
			w.Update(16 * time.Millisecond)
			want := []string{"Steering", "Motion", "Collision", "Reversion"}
			if len(order) != len(want) {
				t.Fatalf("run %d: expected %d systems, got %d", run, len(want), len(order))
			}
			for i, v := range want {
				if order[i] != v {
					t.Fatalf("run %d: expected system order %v, got %v", run, want, order)
				}
			}
		}
	})
}

type mockSys1 struct{ onUpdate func() }
func (m *mockSys1) Update(w *World, elapsed time.Duration) error { m.onUpdate(); return nil }

type mockSys2 struct{ onUpdate func() }
func (m *mockSys2) Update(w *World, elapsed time.Duration) error { m.onUpdate(); return nil }

type mockSys3 struct{ onUpdate func() }
func (m *mockSys3) Update(w *World, elapsed time.Duration) error { m.onUpdate(); return nil }

type mockSys4 struct{ onUpdate func() }
func (m *mockSys4) Update(w *World, elapsed time.Duration) error { m.onUpdate(); return nil }

