package models

import (
	"api-server/utils"
	"time"
)

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

type TasksFilter struct {
	Query      *string `form:"q" json:"q"`
	DueDateStr *string `form:"due_date" json:"dues_date" binding:"omitempty,dayFormat"`
	Status     *string `form:"status" json:"status" binding:"omitempty,taskStatus"`
}

func (tf TasksFilter) DueDate() *time.Time {
	if tf.DueDateStr == nil {
		return nil
	}
	date, _ := time.Parse(utils.DayDateFmt, *tf.DueDateStr)
	return &date
}
