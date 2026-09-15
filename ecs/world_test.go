package ecs

import (
	"errors"
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
		w.Add(e, kids)
		w.Add(e, Expiry{5 * time.Second})
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
}
