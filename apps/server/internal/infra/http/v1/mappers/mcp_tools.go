package mappers

import (
	"encoding/json"
	"fmt"
	"strings"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
)

func McpServerDTOFromDomain(item domain.MCPServerConfig) dto.McpServerDTO {
	return dto.McpServerDTO{ID: item.ID, ProjectID: item.ProjectID, Name: item.Name, Transport: item.Transport, Status: item.Status, Enabled: item.Enabled, Command: item.Command, Args: CopyStringSliceV1(item.Args), URL: item.URL, Headers: CopyStringMapV1(item.Headers), SecretHeadersHint: SecretKeys(item.SecretHeaders), Env: CopyStringMapV1(item.Env), SecretEnvHint: SecretKeys(item.SecretEnv), TimeoutSec: item.TimeoutSec, Metadata: CopyStringMapV1(item.Metadata), LastSeenAt: item.LastSeenAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func McpServerDTOsFromDomain(items []domain.MCPServerConfig) []dto.McpServerDTO {
	servers := make([]dto.McpServerDTO, 0, len(items))
	for _, item := range items {
		servers = append(servers, McpServerDTOFromDomain(item))
	}
	return servers
}

func McpServerRequestToDomain(input dto.McpServerRequestDTO) domain.MCPServerConfig {
	return domain.MCPServerConfig{ID: input.ID, ProjectID: input.ProjectID, Name: input.Name, Transport: input.Transport, Status: input.Status, Enabled: input.Enabled, Command: input.Command, Args: CopyStringSliceV1(input.Args), URL: input.URL, Headers: CopyStringMapV1(input.Headers), SecretHeaders: CopyStringMapV1(input.SecretHeaders), Env: CopyStringMapV1(input.Env), SecretEnv: CopyStringMapV1(input.SecretEnv), TimeoutSec: input.TimeoutSec, Metadata: CopyStringMapV1(input.Metadata)}
}

func ToolDefinitionDTOFromDomain(item domain.ToolDefinition) dto.ToolDefinitionDTO {
	return dto.ToolDefinitionDTO{ID: item.ID, ProjectID: item.ProjectID, Name: item.Name, DisplayName: item.DisplayName, Description: item.Description, Kind: item.Kind, Status: item.Status, MCPServerID: item.MCPServerID, SourceID: item.SourceID, SkillID: item.SkillID, InputSchema: CopyAnyMapV1(item.InputSchema), Metadata: CopyStringMapV1(item.Metadata), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func ToolDefinitionDTOsFromDomain(items []domain.ToolDefinition) []dto.ToolDefinitionDTO {
	tools := make([]dto.ToolDefinitionDTO, 0, len(items))
	for _, item := range items {
		tools = append(tools, ToolDefinitionDTOFromDomain(item))
	}
	return tools
}

func ToolInvocationDTOFromDomain(item domain.ToolInvocation) dto.ToolInvocationDTO {
	return dto.ToolInvocationDTO{ID: item.ID, AgentRunID: item.AgentRunID, AgentID: item.AgentID, ProjectID: item.ProjectID, ToolID: item.ToolID, ToolName: item.ToolName, Status: item.Status, Arguments: CopyAnyMapV1(item.Arguments), Result: CopyAnyMapV1(item.Result), Error: item.Error, StartedAt: item.StartedAt, CompletedAt: item.CompletedAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func ToolInvocationDTOsFromDomain(items []domain.ToolInvocation) []dto.ToolInvocationDTO {
	invocations := make([]dto.ToolInvocationDTO, 0, len(items))
	for _, item := range items {
		invocations = append(invocations, ToolInvocationDTOFromDomain(item))
	}
	return invocations
}

func SecretKeys(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

func RawSchemaObject(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, err
	}
	if schema == nil {
		return nil, fmt.Errorf("schema must be a JSON object")
	}
	return schema, nil
}
