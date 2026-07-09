package mappers

import (
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
)

func AppSettingDTOFromDomain(item domain.AppSetting) dto.AppSettingDTO {
	return dto.AppSettingDTO{Scope: item.Scope, Key: item.Key, Value: CopyAnyMapV1(item.Value), UpdatedAt: item.UpdatedAt}
}

func AppSettingDTOsFromDomain(items []domain.AppSetting) []dto.AppSettingDTO {
	settings := make([]dto.AppSettingDTO, 0, len(items))
	for _, item := range items {
		settings = append(settings, AppSettingDTOFromDomain(item))
	}
	return settings
}

func AppSettingDTOToDomain(input dto.AppSettingDTO) domain.AppSetting {
	return domain.AppSetting{Scope: input.Scope, Key: input.Key, Value: CopyAnyMapV1(input.Value), UpdatedAt: input.UpdatedAt}
}

func SystemStatusDTOFromDomain(status domain.SystemStatus) dto.SystemStatusDTO {
	return dto.SystemStatusDTO{Status: status.Status, PostgresConfigured: status.PostgresConfigured, QdrantConfigured: status.QdrantConfigured, ProviderCount: status.ProviderCount, ModelCount: status.ModelCount, PendingJobsCount: status.PendingJobsCount, CheckedAt: status.CheckedAt}
}
