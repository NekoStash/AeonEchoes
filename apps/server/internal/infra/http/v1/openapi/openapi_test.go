package openapi

import (
	"encoding/json"
	"testing"
)

func TestAgentRunStreamEventTypeExcludesHeartbeatComments(t *testing.T) {
	if AgentRunStreamEventType("heartbeat").Valid() {
		t.Fatal("heartbeat comment must not be a business AgentRunStreamEvent type")
	}
	businessTypes := []AgentRunStreamEventType{
		RunStarted,
		ModelResolved,
		ToolStarted,
		ToolCompleted,
		ContentDelta,
		RunCompleted,
		RunFailed,
	}
	for _, eventType := range businessTypes {
		if !eventType.Valid() {
			t.Fatalf("business event type %q must be valid", eventType)
		}
	}
}

func TestAgentRunStreamToolContainsOnlyPublicLifecycleIdentity(t *testing.T) {
	tool := AgentRunStreamTool{CallId: "call-1", Name: "character.search", Status: Started}
	if err := tool.Validate(); err != nil || !tool.Valid() {
		t.Fatalf("valid public tool rejected: %v", err)
	}
	payload, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}
	if string(payload) != `{"call_id":"call-1","name":"character.search","status":"started"}` {
		t.Fatalf("public tool payload = %s", payload)
	}
}

func TestAgentRunStreamToolRejectsEmptyPublicIdentity(t *testing.T) {
	payloads := []string{
		`{"call_id":"","name":"character.search","status":"started"}`,
		`{"call_id":"call-1","name":"","status":"completed"}`,
		`{"call_id":"   ","name":"character.search","status":"started"}`,
		`{"call_id":"call-1","name":"   ","status":"completed"}`,
	}
	for _, payload := range payloads {
		var tool AgentRunStreamTool
		if err := json.Unmarshal([]byte(payload), &tool); err != nil {
			t.Fatalf("Unmarshal(%s) error: %v", payload, err)
		}
		if err := tool.Validate(); err == nil || tool.Valid() {
			t.Fatalf("invalid public tool accepted: %s", payload)
		}
	}
}
