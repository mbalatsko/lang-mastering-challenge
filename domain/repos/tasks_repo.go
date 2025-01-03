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

type TasksRepo struct {
	Conn *pgxpool.Pool
}

func NewTasksRepo(conn *pgxpool.Pool) *TasksRepo {
	return &TasksRepo{Conn: conn}
}

func (repo *TasksRepo) ListByUserId(ctx context.Context, userId int, tasksFilter models.TasksFilter) ([]models.TaskData, error) {
	qBuilder := utils.PgxSB.
		Select("id", "name", "due_date", "status", "created_at", "user_id").
		From("tasks").
		Where(sq.Eq{"user_id": userId})

	if tasksFilter.Query != nil && *tasksFilter.Query != "" {
		qBuilder = qBuilder.Where("name like ?", fmt.Sprint("%", *tasksFilter.Query, "%"))
	}

	if tasksFilter.DueDate() != nil {
		dd := *tasksFilter.DueDate()
		fromDueDate := time.Date(dd.Year(), dd.Month(), dd.Day(), 0, 0, 0, 0, time.UTC)
		toDueDate := fromDueDate.Add(24 * time.Hour)
		qBuilder = qBuilder.Where("due_date >= ? and due_date < ?", fromDueDate, toDueDate)
	}

	if tasksFilter.Status != nil {
		qBuilder = qBuilder.Where(sq.Eq{"status": *tasksFilter.Status})
	}

	query, args := qBuilder.MustSql()

	startTime := time.Now()
	tasks, err := pgxutil.Select(ctx, repo.Conn, query, args, pgx.RowToStructByPos[models.TaskData])
	logger.LogDbQueryTime(query, args, err, time.Since(startTime))

	if err != nil {
		return nil, fmt.Errorf("db: failed to query tasks by user id %d: %w", userId, err)
	}

	return tasks, nil
}

func (repo *TasksRepo) GetById(ctx context.Context, id int) (models.TaskData, error) {
	query, args := utils.PgxSB.
		Select("id", "name", "due_date", "status", "created_at", "user_id").
		From("tasks").
		Where(sq.Eq{"id": id}).
		MustSql()

	startTime := time.Now()
	task, err := pgxutil.SelectRow(ctx, repo.Conn, query, args, pgx.RowToStructByPos[models.TaskData])
	logger.LogDbQueryTime(query, args, err, time.Since(startTime))

	if errors.Is(err, pgx.ErrNoRows) {
		return models.TaskData{}, ErrNotFound
	}
	if err != nil {
		return models.TaskData{}, fmt.Errorf("db: failed to query task with ID %d: %w", id, err)
	}

	return task, nil
}

func (repo *TasksRepo) DeleteById(ctx context.Context, id int) error {
	query, args := utils.PgxSB.
		Delete("tasks").
		Where(sq.Eq{"id": id}).
		MustSql()

	startTime := time.Now()
	_, err := pgxutil.ExecRow(ctx, repo.Conn, query, args...)
	logger.LogDbQueryTime(query, args, err, time.Since(startTime))

	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("query: %s args: %d: %w", query, args[0], err)
	}

	return nil
}

func (repo *TasksRepo) UpdateStatus(ctx context.Context, id int, newStatus string) (models.TaskData, error) {
	query, args := utils.PgxSB.
		Update("tasks").
		Set("status", newStatus).
		Where(sq.Eq{"id": id}).
		Suffix("RETURNING id, name, due_date, status, created_at, user_id").
		MustSql()

	startTime := time.Now()
	task, err := pgxutil.SelectRow(ctx, repo.Conn, query, args, pgx.RowToStructByPos[models.TaskData])
	logger.LogDbQueryTime(query, args, err, time.Since(startTime))

	if errors.Is(err, pgx.ErrNoRows) {
		return models.TaskData{}, nil
	}
	if err != nil {
		return models.TaskData{}, fmt.Errorf("db: failed to update task with ID %d: %w", id, err)
	}

	return task, nil
}

func (repo *TasksRepo) Create(ctx context.Context, name string, dueDate *time.Time, userId int) (models.TaskData, error) {
	query, args := utils.PgxSB.
		Insert("tasks").Columns("name", "due_date", "user_id").
		Values(name, dueDate, userId).
		Suffix("RETURNING id, name, due_date, status, created_at, user_id").
		MustSql()

	startTime := time.Now()
	task, err := pgxutil.SelectRow(ctx, repo.Conn, query, args, pgx.RowToStructByPos[models.TaskData])
	logger.LogDbQueryTime(query, args, err, time.Since(startTime))

	if err != nil {
		return models.TaskData{}, fmt.Errorf("db: failed to create task: %w", err)
	}

	return task, nil
}

func (repo *TasksRepo) CreateWithStatus(ctx context.Context, name string, dueDate *time.Time, status string, userId int) (models.TaskData, error) {
	query, args := utils.PgxSB.
		Insert("tasks").Columns("name", "due_date", "status", "user_id").
		Values(name, dueDate, status, userId).
		Suffix("RETURNING id, name, due_date, status, created_at, user_id").
		MustSql()

	startTime := time.Now()
	task, err := pgxutil.SelectRow(ctx, repo.Conn, query, args, pgx.RowToStructByPos[models.TaskData])
	logger.LogDbQueryTime(query, args, err, time.Since(startTime))

	if err != nil {
		return models.TaskData{}, fmt.Errorf("db: failed to create task with status: %w", err)
	}

	return task, nil
}
