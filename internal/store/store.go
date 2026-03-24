package store

import (
	"context"
	"database/sql"
	_ "embed"
	"strings"

	"rustymanager/internal/db"
)

//go:embed migrations.sql
var migrations string

func Migrate(database *sql.DB) error {
	for _, stmt := range strings.Split(migrations, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := database.ExecContext(context.Background(), stmt); err != nil {
			// ALTER TABLE ADD COLUMN fails if the column already exists; that's fine.
			if !strings.Contains(err.Error(), "duplicate column name") {
				return err
			}
		}
	}
	return nil
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
