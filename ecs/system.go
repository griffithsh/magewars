package ecs

import "time"

type System interface {
	Update(*World, time.Duration) error
}
