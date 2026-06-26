package postgres

import (
	"fmt"
	"time"

	"aeonechoes/server/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ repository.AppStore = (*Store)(nil)

// Store implements repository.AppStore using PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) (*Store, error) {
	if pool == nil {
		return nil, fmt.Errorf("postgres pool is nil")
	}
	return &Store{pool: pool}, nil
}

func (s *Store) NewID(prefix string) (string, error) { return newID(prefix) }

func now() time.Time { return time.Now().UTC() }

func isNoRows(err error) bool { return err == pgx.ErrNoRows }

func requireStore(s *Store) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("postgres store is not configured")
	}
	return nil
}
