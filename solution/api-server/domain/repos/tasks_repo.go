package repos

import (
	"api-server/domain/models"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TasksRepo struct {
	Conn *pgxpool.Pool
}

func NewTasksRepo(conn *pgxpool.Pool) *TasksRepo {
	return &TasksRepo{Conn: conn}
}

func (repo *TasksRepo) ListByUserId(ctx context.Context, userId int) ([]models.TaskData, error) {
	rows, err := repo.Conn.Query(
		ctx,
		"SELECT id, name, due_date, status, created_at, user_id FROM tasks WHERE user_id = $1",
		userId,
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, pgx.RowToStructByPos[models.TaskData])
}

func (repo *TasksRepo) GetById(ctx context.Context, id int) (task models.TaskData, found bool, err error) {
	rows, err := repo.Conn.Query(ctx, "SELECT id, name, due_date, status, created_at, user_id FROM tasks WHERE id = $1", id)
	if err != nil {
		return models.TaskData{}, false, err
	}

	task, err = pgx.CollectOneRow(rows, pgx.RowToStructByPos[models.TaskData])
	if errors.Is(err, pgx.ErrNoRows) {
		return models.TaskData{}, false, nil
	}
	if err != nil {
		return models.TaskData{}, false, err
	}
	return task, true, nil
}

func (repo *TasksRepo) DeleteById(ctx context.Context, id int) error {
	_, err := repo.Conn.Exec(ctx, "DELETE FROM tasks WHERE id = $1", id)
	return err
}

func (repo *TasksRepo) UpdateStatus(ctx context.Context, taskId int, newStatus string) (models.TaskData, error) {
	rows, err := repo.Conn.Query(
		ctx,
		"UPDATE tasks SET status = $1 WHERE id = $2 RETURNING id, name, due_date, status, created_at, user_id",
		newStatus,
		taskId,
	)
	if err != nil {
		return models.TaskData{}, err
	}

	return pgx.CollectOneRow(rows, pgx.RowToStructByPos[models.TaskData])
}

func (repo *TasksRepo) Create(ctx context.Context, name string, dueDate *time.Time, userId int) (models.TaskData, error) {
	rows, err := repo.Conn.Query(
		ctx,
		"INSERT INTO tasks (name, due_date, user_id) VALUES ($1, $2, $3) RETURNING id, name, due_date, status, created_at, user_id",
		name,
		dueDate,
		userId,
	)
	if err != nil {
		return models.TaskData{}, err
	}

	return pgx.CollectOneRow(rows, pgx.RowToStructByPos[models.TaskData])
}

func (repo *TasksRepo) CreateWithStatus(ctx context.Context, name string, dueDate *time.Time, status string, userId int) (models.TaskData, error) {
	rows, err := repo.Conn.Query(
		ctx,
		"INSERT INTO tasks (name, due_date, status, user_id) VALUES ($1, $2, $3, $4) RETURNING id, name, due_date, status, created_at, user_id",
		name,
		dueDate,
		status,
		userId,
	)
	if err != nil {
		return models.TaskData{}, err
	}

	return pgx.CollectOneRow(rows, pgx.RowToStructByPos[models.TaskData])
}
