package domain

import (
	"context"

	"meerkat-v0/pkg/utils"
)

// EntityID is re-exported for convenience
type EntityID = utils.EntityID

// ServiceConfig represents a service configuration
type ServiceConfig struct {
	Name string `json:"name"`
}

func (c *ServiceConfig) Valid(ctx context.Context) map[string]string {
	err := utils.CheckName(c.Name)
	if err != nil {
		return map[string]string{
			"name": err.Error(),
		}
	}

	return nil
}

// NewServiceID creates a service entity ID
func NewServiceID(instance string, name string) utils.EntityID {
	return utils.EntityID{
		Kind: "service",
		Labels: map[string]string{
			"instance": instance,
			"name":     name,
		},
	}
}

