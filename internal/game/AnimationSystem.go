package game

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/griffithsh/magewars/ecs"
)

type Frame struct {
	Texture          string
	X, Y             int
	W, H             int
	XOffset, YOffset int
	Duration         time.Duration
}

type Frameset struct {
	frames   []Frame
	progress time.Duration
}

func (f *Frameset) Duration() time.Duration {
	total := time.Duration(0)
	for _, frame := range f.frames {
		total += frame.Duration
	}
	return total
}

// AnimationEndBehavior is an enum that describes what happens when an animation
// finishes all of its frames.
type AnimationEndBehavior int

const (
	// AnimationLoops means that the animation will restart from the first frame
	// after the last frame expires.
	AnimationLoops AnimationEndBehavior = iota
	// HoldLastFrame means that the animation will run through all frames, and
	// then stick on the last frame, removing the AnimationCommand Component.
	HoldLastFrame
	// DestroyEntity means that after the animation is complete, the Entity that
	// owns it is destroyed. Use with caution!
	DestroyEntity
)

// AnimationStart is an enum that describes where in the sequence of frames the
// animation will start at.
type AnimationStart int

const (
	// Beginning means that the animation will animate from the first frame.
	Beginning AnimationStart = iota

	// RandomFrame means that the animation will start its animation cycle from
	// a random point in the animation, and not necessarily at the start of a
	// frame. This is mostly useful for looping animations in backgrounds.
	RandomFrame
)

// AnimationCommand is a Component that can be applied to Entities so that they
// are assigned an animated Sprite.
type AnimationCommand struct {
	// AnimationName instructs which animation this entity will take. The string
	// is the key/id of an "*.anim.csv" in the tar.
	AnimationName string
	OnEnd         AnimationEndBehavior
	Start         AnimationStart
}

type AnimationProvider interface {
	GetAnimation(name string) (val []Frame, ok bool)
}

// AnimationSystem syncronises the Sprite of an Entity to the frames of an
// animation, and handles what happens next when the animation is complete.
type AnimationSystem struct {
	provider AnimationProvider
}

func NewAnimationSystem(provider AnimationProvider) *AnimationSystem {
	return &AnimationSystem{provider: provider}
}

func (s *AnimationSystem) Update(w *ecs.World, t time.Duration) error {
	entities := w.QueryAny(ecs.Ty[AnimationCommand](), ecs.Ty[Frameset]())
	for _, e := range entities {
		ac := ecs.Get[AnimationCommand](w, e)
		fs := ecs.Get[Frameset](w, e)

		// For anything with an AnimationCommand Component but no Frameset
		// Component, initialise a Frameset Component for the entity
		if ac != nil && fs == nil {
			// load the frameset info from archive, check the start command to initialise the frames
			frames, ok := s.provider.GetAnimation(ac.AnimationName)
			if !ok {
				return fmt.Errorf("could not load animation with name %q", ac.AnimationName)
			}

			fs := Frameset{
				frames: frames,
			}
			switch ac.Start {
			case RandomFrame:
				fs.progress = time.Millisecond * time.Duration(rand.Int64N(fs.Duration().Milliseconds()))
			default: // i.e Beginning
				fs.progress = time.Duration(0)
			}
			ecs.Add(w, e, &fs)

			continue
		}

		// For anything with a Frameset Component, but no AnimationCommand Component, remove the Frameset Component
		if ac == nil && fs != nil {
			ecs.Remove[Frameset](w, e)
			continue
		}

		// For anything with both Components, step forward through frames, performing end behavior if triggered
		if ac != nil && fs != nil {
			// Advance animation based on how much time has passed.
			fs.progress += t

			if fs.progress >= fs.Duration() {
				// Trigger OnEnd behaviour
				switch ac.OnEnd {
				case HoldLastFrame:
					// Apply last frame to sprite.
					frame := fs.frames[len(fs.frames)-1]
					sprite := Sprite{
						Texture: frame.Texture,
						X:       frame.X,
						Y:       frame.Y,
						W:       frame.W,
						H:       frame.H,
					}
					ecs.Add(w, e, &sprite)
					ecs.Remove[Frameset](w, e)
					ecs.Remove[AnimationCommand](w, e)
					continue
				case DestroyEntity:
					w.DestroyEntity(e)
					continue
				default: // i.e AnimationLoops
					fs.progress = fs.progress % fs.Duration()
				}
			}
			// Find current frame by walking through frames and accumulating
			// durations until we exceed fs.progress.
			elapsed := time.Duration(0)
			frame := fs.frames[len(fs.frames)-1]
			for _, f := range fs.frames {
				elapsed += f.Duration
				if fs.progress < elapsed {
					frame = f
					break
				}
			}
			sprite := Sprite{
				Texture: frame.Texture,
				X:       frame.X,
				Y:       frame.Y,
				W:       frame.W,
				H:       frame.H,
			}
			ecs.Add(w, e, &sprite)
		}
	}

	return nil
}
