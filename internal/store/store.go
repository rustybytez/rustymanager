package store

import (
	"context"
	"database/sql"
	_ "embed"

	"rustymanager/internal/db"
)

//go:embed migrations.sql
var migrations string

func Migrate(database *sql.DB) error {
	_, err := database.ExecContext(context.Background(), migrations)
	return err
}

type Store struct {
	q db.Querier
}

func New(q db.Querier) *Store {
	return &Store{q: q}
}

func (s *Store) Queries() db.Querier {
	return s.q
}
