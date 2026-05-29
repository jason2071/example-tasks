package model

import "time"

type TaskRequest struct {
	Title    *string `json:"title" validate:"required"`
	Status   *string `json:"status" validate:"required"`
	Priority *int    `json:"priority" validate:"omitempty,min=1,max=5"`
}

type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Priority  *int      `json:"priority"`
}

type Pagination struct {
	NextCursor int64 `json:"next_cursor"`
	PageSize   int   `json:"page_size"`
}

type PagedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}
