package event

import "reflect"

type Bus struct {
	handlers map[reflect.Type][]any
}

func NewBus() *Bus {
	return &Bus{
		handlers: make(map[reflect.Type][]any),
	}
}

func Subscribe[T any](b *Bus, handler func(*T)) {
	t := reflect.TypeOf((*T)(nil))
	b.handlers[t] = append(b.handlers[t], handler)
}

func Publish[T any](b *Bus, event *T) {
	t := reflect.TypeOf(event)
	for _, raw := range b.handlers[t] {
		if handler, ok := raw.(func(*T)); ok {
			handler(event)
		}
	}
}
