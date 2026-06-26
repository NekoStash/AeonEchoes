package repository_test

import (
	"testing"

	"aeonechoes/server/internal/memory"
	"aeonechoes/server/internal/repository"
)

func TestMemoryStoreImplementsAppStore(t *testing.T) {
	var _ repository.AppStore = memory.NewStore()
}
