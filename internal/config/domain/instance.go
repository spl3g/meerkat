package domain

import (
	"context"
	"encoding/json"

	"meerkat-v0/pkg/utils"
)

// InstanceConfig represents the top-level instance configuration
type InstanceConfig struct {
	Name     string            `json:"name"`
	Services []json.RawMessage `json:"services"`
}

func (c *InstanceConfig) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string, 2)

	err := utils.CheckName(c.Name)
	if err != nil {
		problems["name"] = err.Error()
	}

	if len(c.Services) == 0 {
		problems["services"] = "services cannot be empty"
	}

	return problems
}

