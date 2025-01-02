package services

import (
	"api-server/internal/domain/models"
	"api-server/internal/domain/repos"
	"context"
	"errors"
)

var (
	ErrTaskDoesNotExist = errors.New("task with given id does not exist")
	ErrNotOwner         = errors.New("user is not owner of this item")
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

func (s *TasksService) DeleteById(ctx context.Context, taskId int, reqUserId int) error {
	taskDb, found, err := s.Repo.GetById(ctx, taskId)
	if err != nil {
		return nil
	}
	if !found {
		return ErrTaskDoesNotExist
	}

	if taskDb.UserId != reqUserId {
		return ErrNotOwner
	}
	return s.Repo.DeleteById(ctx, taskId)
}

func (s *TasksService) UpdateStatus(ctx context.Context, taskId int, newStatus string, reqUserId int) (models.TaskData, error) {
	taskDb, found, err := s.Repo.GetById(ctx, taskId)
	if err != nil {
		return models.TaskData{}, nil
	}
	if !found {
		return models.TaskData{}, ErrTaskDoesNotExist
	}

	if taskDb.UserId != reqUserId {
		return models.TaskData{}, ErrNotOwner
	}

	return s.Repo.UpdateStatus(ctx, taskId, newStatus)
}
