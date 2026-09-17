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

}
