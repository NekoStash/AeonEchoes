package agent

import (
	"fmt"
	"sort"
	"strings"

	"aeonechoes/server/internal/domain"
)

const (
	// ModelRoutingSettingScope stores explicit model routes in settings. Each key is
	// an agent role name or "embedding", and value.model must be "providerId:modelId".
	ModelRoutingSettingScope    = "model_routing"
	ModelRoutingSettingValueKey = "model"
	ModelRoutingEmbeddingKey    = "embedding"
)

// ModelCatalog is the minimal repository surface needed by model routing.
type ModelCatalog interface {
	ListModelsByKind(kind domain.ModelKind) ([]domain.ModelConfig, error)
	GetProvider(id string) (domain.ProviderConfig, error)
	ListSettings(scope string) ([]domain.AppSetting, error)
}

// ModelSelection pairs a model with its provider configuration and route metadata.
type ModelSelection struct {
	Model            domain.ModelConfig
	Provider         domain.ProviderConfig
	RouteKey         string
	ResolutionSource string
}

// ModelRouter deterministically selects enabled text or embedding models.
type ModelRouter struct {
	catalog ModelCatalog
	roles   *AgentRoleRegistry
}

func NewModelRouter(catalog ModelCatalog, roles *AgentRoleRegistry) *ModelRouter {
	return &ModelRouter{catalog: catalog, roles: roles}
}

func (r *ModelRouter) SelectTextModel(role domain.AgentRole) (ModelSelection, error) {
	if r == nil || r.catalog == nil {
		return ModelSelection{}, fmt.Errorf("model router is not configured")
	}
	routeKey := string(role)
	if r.roles != nil {
		if _, err := r.roles.Get(role); err != nil {
			return ModelSelection{}, err
		}
	}
	models, err := r.catalog.ListModelsByKind(domain.ModelKindText)
	if err != nil {
		return ModelSelection{}, err
	}
	if reference, ok, err := r.explicitModelReference(routeKey); err != nil {
		return ModelSelection{}, err
	} else if ok {
		return r.selectExplicitModel(models, domain.ModelKindText, routeKey, reference)
	}
	models = filterRoleModels(models, role)
	return r.selectModel(models, domain.ModelKindText, routeKey, "role_fallback")
}

func (r *ModelRouter) SelectEmbeddingModel() (ModelSelection, error) {
	if r == nil || r.catalog == nil {
		return ModelSelection{}, fmt.Errorf("model router is not configured")
	}
	models, err := r.catalog.ListModelsByKind(domain.ModelKindEmbedding)
	if err != nil {
		return ModelSelection{}, err
	}
	if reference, ok, err := r.explicitModelReference(ModelRoutingEmbeddingKey); err != nil {
		return ModelSelection{}, err
	} else if ok {
		return r.selectExplicitModel(models, domain.ModelKindEmbedding, ModelRoutingEmbeddingKey, reference)
	}
	return r.selectModel(models, domain.ModelKindEmbedding, ModelRoutingEmbeddingKey, "kind_fallback")
}

func (r *ModelRouter) explicitModelReference(routeKey string) (string, bool, error) {
	settings, err := r.catalog.ListSettings(ModelRoutingSettingScope)
	if err != nil {
		return "", false, fmt.Errorf("list %q settings for model routing: %w", ModelRoutingSettingScope, err)
	}
	for _, setting := range settings {
		if setting.Key != routeKey {
			continue
		}
		reference, ok, err := modelReferenceFromSetting(setting)
		if err != nil {
			return "", false, err
		}
		if !ok {
			return "", false, nil
		}
		return reference, true, nil
	}
	return "", false, nil
}

func modelReferenceFromSetting(setting domain.AppSetting) (string, bool, error) {
	if len(setting.Value) == 0 {
		return "", false, nil
	}
	raw, ok := setting.Value[ModelRoutingSettingValueKey]
	if !ok {
		raw, ok = setting.Value["value"]
	}
	if !ok {
		return "", false, nil
	}
	reference, ok := raw.(string)
	if !ok {
		return "", false, fmt.Errorf("model routing setting %q/%q %q must be a string", setting.Scope, setting.Key, ModelRoutingSettingValueKey)
	}
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", false, nil
	}
	providerID, modelID, err := splitModelReference(reference)
	if err != nil {
		return "", false, fmt.Errorf("model routing setting %q/%q is invalid: %w", setting.Scope, setting.Key, err)
	}
	return providerID + ":" + modelID, true, nil
}

func splitModelReference(reference string) (string, string, error) {
	providerID, modelID, ok := strings.Cut(strings.TrimSpace(reference), ":")
	providerID = strings.TrimSpace(providerID)
	modelID = strings.TrimSpace(modelID)
	if !ok || providerID == "" || modelID == "" {
		return "", "", fmt.Errorf("model reference must use providerId:modelId format")
	}
	return providerID, modelID, nil
}

func (r *ModelRouter) selectExplicitModel(models []domain.ModelConfig, kind domain.ModelKind, routeKey, reference string) (ModelSelection, error) {
	providerID, modelID, err := splitModelReference(reference)
	if err != nil {
		return ModelSelection{}, err
	}
	for _, model := range models {
		if model.ProviderID != providerID {
			continue
		}
		if model.Name == modelID || model.ID == reference || model.ID == modelID {
			return r.selectionForModel(model, routeKey, "explicit_setting")
		}
	}
	return ModelSelection{}, fmt.Errorf("explicit model route %q references %q, but no enabled %s model is configured for provider %q and model %q", routeKey, reference, kind, providerID, modelID)
}

func (r *ModelRouter) selectModel(models []domain.ModelConfig, kind domain.ModelKind, routeKey, resolutionSource string) (ModelSelection, error) {
	if len(models) == 0 {
		return ModelSelection{}, fmt.Errorf("no enabled %s model is configured", kind)
	}
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].DefaultForKind != models[j].DefaultForKind {
			return models[i].DefaultForKind
		}
		if models[i].RoutingWeight != models[j].RoutingWeight {
			return models[i].RoutingWeight > models[j].RoutingWeight
		}
		return models[i].CreatedAt.Before(models[j].CreatedAt)
	})
	return r.selectionForModel(models[0], routeKey, resolutionSource)
}

func (r *ModelRouter) selectionForModel(model domain.ModelConfig, routeKey, resolutionSource string) (ModelSelection, error) {
	providerCfg, err := r.catalog.GetProvider(model.ProviderID)
	if err != nil {
		return ModelSelection{}, err
	}
	if !providerCfg.Enabled {
		return ModelSelection{}, fmt.Errorf("selected model %q belongs to disabled provider %q", model.ID, providerCfg.ID)
	}
	return ModelSelection{Model: model, Provider: providerCfg, RouteKey: routeKey, ResolutionSource: resolutionSource}, nil
}

func filterRoleModels(models []domain.ModelConfig, role domain.AgentRole) []domain.ModelConfig {
	filtered := make([]domain.ModelConfig, 0, len(models))
	for _, model := range models {
		if len(model.AllowedAgentRoles) == 0 {
			filtered = append(filtered, model)
			continue
		}
		for _, allowed := range model.AllowedAgentRoles {
			if allowed == role {
				filtered = append(filtered, model)
				break
			}
		}
	}
	return filtered
}
