package main

import (
	"flag"
	"fmt"
	"log"

	_ "github.com/TranVinhHien/ticket-date/be/inventory-service/docs"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/config"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database/migrations"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/repository"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/router"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/services"
)

var confPath = flag.String("config", "", "Path to inventory service config yaml file")

// @title Inventory Service API
// @version 1.0
// @description Inventory Service API for ticket date system.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api
func main() {
	flag.Parse()
	targetConfig := config.ResolveConfigFile(*confPath)
	conf, err := config.LoadConfig(targetConfig)
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	config.App = *conf

	// Connect to database (pgxpool)
	database.Connect()
	defer database.Close()

	pool := database.GetPool()
	err = database.Migrate(pool, migrations.FS, ".")
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	fmt.Println("Database migrated successfully")

	// Initialize repositories
	inventoryRepo := repository.NewInventoryRepository(pool)

	// Initialize services
	inventoryService := services.NewInventoryService(inventoryRepo)

	// Initialize handlers
	inventoryHandler := handlers.NewInventoryHandler(inventoryService)

	// Setup Router
	r := router.SetupRouter(inventoryHandler)

	// Start server
	port := config.App.HttpServer.Port
	if port == 0 {
		port = 8080
	}
	address := fmt.Sprintf(":%d", port)
	log.Printf("Server starting on %s", address)
	if err := r.Run(address); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
