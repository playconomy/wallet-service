package database

import (
	"database/sql"

	"github.com/playconomy/wallet-service/internal/config"

	_ "github.com/lib/pq"
	"go.uber.org/fx"
)

// Module provides database dependencies
var Module = fx.Options(
	fx.Provide(NewConnection),
)

func NewConnection(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Database.GetDSN())
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}
