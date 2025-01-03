package utils

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var PgxSB = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func TruncateTables(conn *pgxpool.Pool, tables []string) {
	batch := &pgx.Batch{}
	for _, t := range tables {
		batch.Queue(fmt.Sprintf("DELETE FROM %s", t))
	}
	err := conn.SendBatch(context.Background(), batch).Close()
	if err != nil {
		panic(err)
	}
}
