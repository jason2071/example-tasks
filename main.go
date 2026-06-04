package main

import (
	"database/sql"
	"fmt"
	"log"

	"example-tasks/internal/config"
	"example-tasks/internal/handler"
	"example-tasks/internal/model"
	"example-tasks/internal/repository"
	"example-tasks/internal/router"
	"example-tasks/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// โหลด config จาก config.yaml + env variables
	appConfig, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db, err := newDB(appConfig.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// repository -> service -> handler (manual DI)
	taskRepo := repository.NewTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)
	healthSvc := service.NewHealthService(db)

	taskHandler := handler.NewTaskHandler(taskService)
	healthHandler := handler.NewHealthHandler(healthSvc, appConfig.AppInfo)

	// server
	app := gin.Default()
	router.Setup(app, taskHandler, healthHandler)

	log.Fatal(app.Run(":3000"))
}

// newDB opens and verifies the PostgreSQL connection.
// NOTE: sslmode is hardcoded to disable; config SSLMode is intentionally ignored.
func newDB(cfg model.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	fmt.Println("Successfully connected to the database!")
	return db, nil
}
