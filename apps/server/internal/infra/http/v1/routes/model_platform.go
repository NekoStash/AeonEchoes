package routes

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/infra/http/v1/respond"
)

func (s *Router) v1ListProviders(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListProviders()
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.ProviderDTOsFromDomain(items), len(items), 0)
}

func (s *Router) v1CreateProvider(w http.ResponseWriter, r *http.Request) {
	var input dto.ProviderRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	created, err := s.store.CreateProvider(mappers.ProviderRequestToDomain(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusCreated, mappers.ProviderDTOFromDomain(created))
}

func (s *Router) v1GetProvider(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetProvider(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.ProviderDTOFromDomain(item))
}

func (s *Router) v1UpdateProvider(w http.ResponseWriter, r *http.Request) {
	var input dto.ProviderRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	id := r.PathValue("id")
	existing, err := s.store.GetProvider(id)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	updated, err := s.store.UpdateProvider(id, mappers.ApplyProviderRequest(input, existing))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.ProviderDTOFromDomain(updated))
}

func (s *Router) v1DeleteProvider(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteProvider(r.PathValue("id")); err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Router) v1RefreshProviderModels(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.GetProvider(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	client, err := s.providers.NewModelListClient(cfg)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	infos, err := client.ListModels(r.Context())
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadGateway, err)
		return
	}
	models := make([]domain.ModelConfig, 0, len(infos))
	now := time.Now().UTC()
	for _, info := range infos {
		id := fmt.Sprintf("%s:%s", cfg.ID, info.ID)
		discovered := mappers.DiscoveredModelConfig(cfg, info, now)
		model, err := s.store.GetModel(id)
		if err == nil {
			model = mappers.MergeDiscoveredModel(model, discovered, info)
			model, err = s.store.UpdateModel(id, model)
		} else if strings.Contains(err.Error(), "not found") {
			model, err = s.store.CreateModel(discovered)
		}
		if err != nil {
			respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
			return
		}
		models = append(models, model)
	}
	if err := s.store.TouchProviderModelRefresh(cfg.ID); err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	refreshedProvider, err := s.store.GetProvider(cfg.ID)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, dto.ProviderModelRefreshDTO{Models: mappers.ModelDTOsFromDomain(models), Count: len(models), Provider: mappers.ProviderDTOFromDomain(refreshedProvider)})
}

func (s *Router) v1ListModels(w http.ResponseWriter, r *http.Request) {
	kind := domain.ModelKind(r.URL.Query().Get("kind"))
	var (
		items []domain.ModelConfig
		err   error
	)
	if kind != "" {
		if !kind.Valid() {
			respond.Error(w, r, http.StatusBadRequest, "bad_request", fmt.Sprintf("invalid model kind %q", kind), nil)
			return
		}
		items, err = s.store.ListModelsByKind(kind)
	} else {
		items, err = s.store.ListModels()
	}
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.ModelDTOsFromDomain(items), len(items), 0)
}

func (s *Router) v1CreateModel(w http.ResponseWriter, r *http.Request) {
	var input dto.ModelRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	created, err := s.store.CreateModel(mappers.ModelRequestToDomain(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusCreated, mappers.ModelDTOFromDomain(created))
}

func (s *Router) v1GetModel(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetModel(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.ModelDTOFromDomain(item))
}

func (s *Router) v1UpdateModel(w http.ResponseWriter, r *http.Request) {
	var input dto.ModelRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	updated, err := s.store.UpdateModel(r.PathValue("id"), mappers.ModelRequestToDomain(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.ModelDTOFromDomain(updated))
}

func (s *Router) v1DeleteModel(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteModel(r.PathValue("id")); err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Router) v1GetModelRouting(w http.ResponseWriter, r *http.Request) {
	routes, err := s.currentModelRoutingRoutes()
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, dto.ModelRoutingDTO{Routes: routes})
}

func (s *Router) v1PutModelRouting(w http.ResponseWriter, r *http.Request) {
	var input dto.ModelRoutingDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	if input.Routes == nil {
		respond.Error(w, r, http.StatusBadRequest, "bad_request", "model routing routes must not be null", nil)
		return
	}
	for routeKey, reference := range input.Routes {
		routeKey = strings.TrimSpace(routeKey)
		reference = strings.TrimSpace(reference)
		if routeKey == "" {
			respond.Error(w, r, http.StatusBadRequest, "bad_request", "model routing route key must not be empty", nil)
			return
		}
		if reference != "" {
			if err := s.ensureModelReferenceExists(reference); err != nil {
				respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
				return
			}
		}
		if _, err := s.store.UpsertSetting(domain.AppSetting{Scope: agent.ModelRoutingSettingScope, Key: routeKey, Value: map[string]any{agent.ModelRoutingSettingValueKey: reference}}); err != nil {
			respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
			return
		}
	}
	routes, err := s.currentModelRoutingRoutes()
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, dto.ModelRoutingDTO{Routes: routes})
}

func (s *Router) currentModelRoutingRoutes() (map[string]string, error) {
	settings, err := s.store.ListSettings(agent.ModelRoutingSettingScope)
	if err != nil {
		return nil, err
	}
	routes := map[string]string{}
	for _, setting := range settings {
		if len(setting.Value) == 0 {
			continue
		}
		raw, ok := setting.Value[agent.ModelRoutingSettingValueKey]
		if !ok {
			raw, ok = setting.Value["value"]
		}
		if !ok || raw == nil {
			continue
		}
		reference, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("model routing setting %q/%q model must be a string", setting.Scope, setting.Key)
		}
		routes[setting.Key] = reference
	}
	return routes, nil
}

func (s *Router) ensureModelReferenceExists(reference string) error {
	providerID, modelID, err := splitV1ModelReference(reference)
	if err != nil {
		return err
	}
	models, err := s.store.ListModels()
	if err != nil {
		return err
	}
	for _, model := range models {
		if model.ID == reference {
			return nil
		}
		if model.ProviderID == providerID && (model.Name == modelID || model.ID == modelID) {
			return nil
		}
	}
	return fmt.Errorf("model routing reference %q does not match any configured model", reference)
}

func splitV1ModelReference(reference string) (string, string, error) {
	providerID, modelID, ok := strings.Cut(strings.TrimSpace(reference), ":")
	providerID = strings.TrimSpace(providerID)
	modelID = strings.TrimSpace(modelID)
	if !ok || providerID == "" || modelID == "" {
		return "", "", fmt.Errorf("model reference must use providerId:modelId format")
	}
	return providerID, modelID, nil
}
