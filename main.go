package main

import (
	"database/sql"
	"fmt"
	"log"

	"example-tasks/config"
	"example-tasks/handler"
	"example-tasks/repository"
	"example-tasks/service"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
)

func main() {

	// โหลด config จาก config.yaml + env variables
	appConfig, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db_cfg := appConfig.Database

	fmt.Printf("Connecting to host: %s, database: %s, user: %s\n",
		db_cfg.Host,
		db_cfg.Database,
		db_cfg.User,
	)

	// database connection
	pqInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable", db_cfg.Host, db_cfg.Port, db_cfg.User, db_cfg.Password, db_cfg.Database)

	db, err := sql.Open("postgres", pqInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Successfully connected to the database!")

	// repository
	taskRepo := repository.NewTaskRepository(db)

	// service
	taskService := service.NewTaskService(taskRepo)
	healthSvc := service.NewHealthService(db)

	// handler
	taskHandler := handler.NewTaskHandler(taskService)
	healthHandler := handler.NewHealthHandler(healthSvc, appConfig.AppInfo)

	// server
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Get("/live", healthHandler.Live)
	app.Get("/ready", healthHandler.Ready)
	app.Get("/info", healthHandler.Info)

	app.Post("/task", taskHandler.CreateTask)
	app.Get("/tasks", taskHandler.GetTasks)
	app.Get("/task/:id", taskHandler.GetTaskByID)
	app.Patch("/task/:id", taskHandler.UpdateTask)
	app.Delete("/task/:id", taskHandler.DeleteTask)

	log.Fatal(app.Listen(":3000"))
}
