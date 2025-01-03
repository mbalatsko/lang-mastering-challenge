package db

import (
	"api-server/utils"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

func ConnectDB() *pgxpool.Pool {
	var (
		dbHost     = utils.MustGetenv("PG_HOST")
		dbPort     = utils.MustGetenv("PG_PORT")
		dbUser     = utils.MustGetenv("PG_USER")
		dbPassword = utils.MustGetenv("PG_PASSWORD")
		dbName     = utils.MustGetenv("DB_NAME")
	)

	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", dbUser, dbPassword, dbHost, dbPort, dbName)

	conn, err := pgxpool.New(context.Background(), url)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %s", err.Error()))
	}

	log.WithFields(
		log.Fields{
			"url": url,
		},
	).Info("Successfully connected to database")
	return conn
}
