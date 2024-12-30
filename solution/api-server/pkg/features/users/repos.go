package users

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserData struct {
	Id           int
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

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

func (repo *UserRepo) Create(ctx context.Context, email string, passwordHash string) error {
	_, err := repo.Conn.Exec(ctx, "INSERT INTO users (email, password_hash) VALUES ($1, $2)", email, passwordHash)
	return err
}

func (repo *UserRepo) GetByEmail(ctx context.Context, email string) (user UserData, found bool, err error) {
	rows, err := repo.Conn.Query(ctx, "SELECT id, email, password_hash, created_at FROM users WHERE email = $1", email)
	if err != nil {
		return UserData{}, false, err
	}

	user, err = pgx.CollectOneRow(rows, pgx.RowToStructByPos[UserData])
	if errors.Is(err, pgx.ErrNoRows) {
		return UserData{}, false, nil
	}
	if err != nil {
		return UserData{}, false, err
	}
	return user, true, nil
}
