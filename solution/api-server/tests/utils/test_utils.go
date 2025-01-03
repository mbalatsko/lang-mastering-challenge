package test_utils

import (
	"api-server/domain/models"
	"api-server/domain/repos"
	"context"
)

func Map[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}

func MapTasksToName(tasks []models.TaskData) []string {
	return Map(tasks, func(t models.TaskData) string { return t.Name })
}

func CreateUserWithTasks(
	userCred models.UserRegister,
	tasksData []models.TaskData,
	userRepo *repos.UsersRepo,
	tasksRepo *repos.TasksRepo,
) (models.UserData, []models.TaskData) {
	createdTasks := make([]models.TaskData, 0, len(tasksData))
	user, _ := userRepo.Create(context.Background(), userCred.Email, userCred.Password)
	for _, t := range tasksData {
		createdTask, err := tasksRepo.CreateWithStatus(context.Background(), t.Name, t.DueDate, t.Status, user.Id)
		if err != nil {
			panic(err)
		}
		createdTasks = append(createdTasks, createdTask)
	}
	return user, createdTasks
}
