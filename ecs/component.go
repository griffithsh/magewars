package ecs

type ComponentName string

type Component interface {
	Name() ComponentName
}
