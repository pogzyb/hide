package infra

import "context"

type Provider interface {
	Deploy(context.Context, map[string]string) (HideInstanceInfo, error)
	Destroy(context.Context, map[string]string) error
	CheckExisting(context.Context) error
}

type HideInstanceInfo struct {
	Hostname string
	UID string
}