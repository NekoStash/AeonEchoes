package openapi

import (
	"fmt"
	"strings"
)

// Valid reports whether the generated public tool lifecycle payload satisfies
// the non-empty identity and status constraints declared by OpenAPI.
func (t AgentRunStreamTool) Valid() bool {
	return t.Validate() == nil
}

// Validate enforces constraints that oapi-codegen model generation does not
// currently materialize from string minLength keywords.
func (t AgentRunStreamTool) Validate() error {
	if strings.TrimSpace(t.CallId) == "" {
		return fmt.Errorf("agent run stream tool call_id must not be empty")
	}
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("agent run stream tool name must not be empty")
	}
	if !t.Status.Valid() {
		return fmt.Errorf("agent run stream tool status %q is invalid", t.Status)
	}
	return nil
}
