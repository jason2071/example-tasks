package service

import (
	"context"
	"database/sql"
)

type HealthService interface {
	Ping(ctx context.Context) error
}

type healthServiceImpl struct {
	db *sql.DB
}

func NewHealthService(db *sql.DB) HealthService {
	return &healthServiceImpl{db: db}
}

func (s *healthServiceImpl) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
