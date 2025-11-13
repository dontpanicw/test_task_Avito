package migrations

import (
	"context"
	"database/sql"
	"embed"
	"github.com/pressly/goose/v3"
)

//go:embed *.sql

var embedMigrations embed.FS

func Migrate(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)
	goose.SetDialect("postgres")

	// "." означает: использовать файлы .sql из той же директории, что и migrate.go
	return goose.UpContext(context.Background(), db, ".")
}
