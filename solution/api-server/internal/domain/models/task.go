package models

import "time"

var ValidTaskStatuses = []string{
	"Won't do",
	"To do",
	"In progress",
	"Done",
}

type TaskStatus struct {
	Status string `json:"status" binding:"required,taskStatus"`
}

type TaskCreate struct {
	Name    string     `json:"name" binding:"required"`
	DueDate *time.Time `json:"due_date"`
}

type TaskData struct {
	Id        int        `json:"id"`
	Name      string     `json:"name"`
	DueDate   *time.Time `json:"due_date"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UserId    int        `json:"-"`
}
