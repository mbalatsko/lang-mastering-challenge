package services

import (
	"api-server/internal/domain/models"
	"api-server/internal/domain/repos"
	"context"
)

type TasksService struct {
	Repo *repos.TasksRepo
}

func NewTasksService(repo *repos.TasksRepo) *TasksService {
	return &TasksService{Repo: repo}
}

func (s *TasksService) Create(ctx context.Context, task models.TaskCreate, userId int) (models.TaskData, error) {
	return s.Repo.Create(ctx, task.Name, task.DueDate, userId)
}

func (s *TasksService) ListByUserId(ctx context.Context, userId int) ([]models.TaskData, error) {
	return s.Repo.ListByUserId(ctx, userId)
}
