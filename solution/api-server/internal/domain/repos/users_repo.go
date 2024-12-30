package repos

import (
	"api-server/internal/domain/models"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	Conn *pgxpool.Pool
}

func NewUserRepo(conn *pgxpool.Pool) *UserRepo {
	return &UserRepo{Conn: conn}
}

func (repo *UserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var emailExists bool

	err := repo.Conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&emailExists)
	if err != nil {
		return false, err
	}
	return emailExists, nil
}

func (repo *UserRepo) Create(ctx context.Context, email string, passwordHash string) (models.UserData, error) {
	rows, err := repo.Conn.Query(
		ctx,
		"INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, password_hash, created_at",
		email,
		passwordHash,
	)
	if err != nil {
		return models.UserData{}, err
	}

	return pgx.CollectOneRow(rows, pgx.RowToStructByPos[models.UserData])
}

func (repo *UserRepo) GetByEmail(ctx context.Context, email string) (user models.UserData, found bool, err error) {
	rows, err := repo.Conn.Query(ctx, "SELECT id, email, password_hash, created_at FROM users WHERE email = $1", email)
	if err != nil {
		return models.UserData{}, false, err
	}

	user, err = pgx.CollectOneRow(rows, pgx.RowToStructByPos[models.UserData])
	if errors.Is(err, pgx.ErrNoRows) {
		return models.UserData{}, false, nil
	}
	if err != nil {
		return models.UserData{}, false, err
	}
	return user, true, nil
}
