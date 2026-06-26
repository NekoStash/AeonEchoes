package agent

import (
	"fmt"

	"aeonechoes/server/internal/domain"
)

// RoleSpec declares the model and tool expectations for a logical agent role.
type RoleSpec struct {
	Role        domain.AgentRole `json:"role"`
	Kind        domain.ModelKind `json:"kind"`
	Description string           `json:"description"`
	Tools       []string         `json:"tools"`
}

// AgentRoleRegistry keeps routing metadata for logical writing roles.
type AgentRoleRegistry struct {
	roles map[domain.AgentRole]RoleSpec
}

func NewAgentRoleRegistry() *AgentRoleRegistry {
	roles := map[domain.AgentRole]RoleSpec{
		domain.AgentRoleGenesisOptimizer: {Role: domain.AgentRoleGenesisOptimizer, Kind: domain.ModelKindText, Description: "turn a project seed into a coherent story bible", Tools: []string{"graph.expand"}},
		domain.AgentRolePlotArchitect:    {Role: domain.AgentRolePlotArchitect, Kind: domain.ModelKindText, Description: "plan arcs, chapters and narrative promises", Tools: []string{"graph.expand"}},
		domain.AgentRoleWorldBuilder:     {Role: domain.AgentRoleWorldBuilder, Kind: domain.ModelKindText, Description: "maintain setting, rules and locations", Tools: []string{"graph.expand"}},
		domain.AgentRoleCharacterKeeper:  {Role: domain.AgentRoleCharacterKeeper, Kind: domain.ModelKindText, Description: "maintain character continuity", Tools: []string{"graph.expand"}},
		domain.AgentRoleContinuityAudit:  {Role: domain.AgentRoleContinuityAudit, Kind: domain.ModelKindText, Description: "audit continuity against facts", Tools: []string{"graph.expand"}},
		domain.AgentRoleWriter:           {Role: domain.AgentRoleWriter, Kind: domain.ModelKindText, Description: "draft prose from context packs", Tools: []string{"graph.expand"}},
		domain.AgentRoleEditor:           {Role: domain.AgentRoleEditor, Kind: domain.ModelKindText, Description: "revise prose while preserving canon", Tools: []string{"graph.expand"}},
		domain.AgentRoleFactExtractor:    {Role: domain.AgentRoleFactExtractor, Kind: domain.ModelKindText, Description: "extract atomic facts after content changes", Tools: []string{"graph.expand"}},
		domain.AgentRoleGraphCurator:     {Role: domain.AgentRoleGraphCurator, Kind: domain.ModelKindText, Description: "refresh graph relations after extraction", Tools: []string{"graph.expand"}},
	}
	return &AgentRoleRegistry{roles: roles}
}

func (r *AgentRoleRegistry) Get(role domain.AgentRole) (RoleSpec, error) {
	if r == nil {
		return RoleSpec{}, fmt.Errorf("agent role registry is nil")
	}
	spec, ok := r.roles[role]
	if !ok {
		return RoleSpec{}, fmt.Errorf("agent role %q is not registered", role)
	}
	return spec, nil
}

func (r *AgentRoleRegistry) List() []RoleSpec {
	items := make([]RoleSpec, 0, len(r.roles))
	for _, spec := range r.roles {
		items = append(items, spec)
	}
	return items
}
