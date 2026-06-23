package configs

import (
	"context"
	"fmt"
	"sync"

	"github.com/masudur-rahman/kazi-ancestry/infra/logr"
	"github.com/masudur-rahman/kazi-ancestry/models"
	"github.com/masudur-rahman/kazi-ancestry/services/all"

	"github.com/masudur-rahman/styx"
	isql "github.com/masudur-rahman/styx/sql"
	"github.com/masudur-rahman/styx/sql/postgres"
	pglib "github.com/masudur-rahman/styx/sql/postgres/lib"

	_ "github.com/lib/pq"
)

// tables managed by Sync, in dependency order.
var tables = []any{models.Person{}, models.User{}, models.Suggestion{}}

var (
	sqlDB isql.Engine
	dbMu  sync.Mutex
)

// GetUnitOfWork returns a UnitOfWork wrapping the active database engine.
func GetUnitOfWork() styx.UnitOfWork {
	dbMu.Lock()
	defer dbMu.Unlock()
	return styx.UnitOfWork{SQL: sqlDB}
}

// InitiateDatabaseConnection opens Postgres, syncs the schema, and wires services.
func InitiateDatabaseConnection(ctx context.Context) error {
	conn, err := pglib.GetPostgresConnection(KaziConfig.Database.Postgres)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	db := postgres.NewPostgres(conn)

	dbMu.Lock()
	sqlDB = db
	dbMu.Unlock()

	if err := db.Sync(ctx, tables...); err != nil {
		return fmt.Errorf("sync schema: %w", err)
	}

	all.InitiateSQLServices(GetUnitOfWork(), logr.DefaultLogger)
	return nil
}
