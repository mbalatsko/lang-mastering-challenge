package repos

import (
	"api-server/app/logger"
	"api-server/domain/models"
	"api-server/utils"
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgxutil"
)

type UsersRepo struct {
	Conn *pgxpool.Pool
}

func NewUsersRepo(conn *pgxpool.Pool) *UsersRepo {
	return &UsersRepo{Conn: conn}
}

func (repo *UsersRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	query, args := utils.PgxSB.
		Select("1").
		Prefix("SELECT EXISTS (").
		From("users").
		Where(sq.Eq{"email": email}).
		Suffix(")").
		MustSql()

	startTime := time.Now()
	var emailExists bool
	err := repo.Conn.QueryRow(ctx, query, args...).Scan(&emailExists)
	logger.LogDbQueryTime(query, args, err, time.Since(startTime))

	if err != nil {
		return false, err
	}
	return emailExists, nil
}

func (repo *UsersRepo) Create(ctx context.Context, email string, passwordHash string) (models.UserData, error) {
	query, args := utils.PgxSB.
		Insert("users").Columns("email", "password_hash").
		Values(email, passwordHash).
		Suffix("RETURNING id, email, password_hash, created_at").
		MustSql()

	startTime := time.Now()
	user, err := pgxutil.SelectRow(ctx, repo.Conn, query, args, pgx.RowToStructByPos[models.UserData])
	logger.LogDbQueryTime(query, args, err, time.Since(startTime))

	if err != nil {
		return models.UserData{}, fmt.Errorf("db: failed to create user: %w", err)
	}
	return user, nil
}

func (repo *UsersRepo) GetByEmail(ctx context.Context, email string) (models.UserData, error) {
	query, args := utils.PgxSB.
		Select("id", "email", "password_hash", "created_at").
		From("users").
		Where(sq.Eq{"email": email}).
		MustSql()

	startTime := time.Now()
	user, err := pgxutil.SelectRow(ctx, repo.Conn, query, args, pgx.RowToStructByPos[models.UserData])
	logger.LogDbQueryTime(query, args, err, time.Since(startTime))

	if errors.Is(err, pgx.ErrNoRows) {
		return models.UserData{}, ErrNotFound
	}
	if err != nil {
		return models.UserData{}, fmt.Errorf("db: failed to query user with email %s: %w", email, err)
	}

	return user, nil
}
