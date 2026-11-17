package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/TranVinhHien/ticket-date/be/event-service/internal/config"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/database"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/database/migrations"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/repository"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/router"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/services"
)

var confPath = flag.String("config", "", "Path to event service config yaml file")

func main() {
	flag.Parse()
	targetConfig := config.ResolveConfigFile(*confPath)
	conf, err := config.LoadConfig(targetConfig)
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	config.App = *conf

	// Connect to database
	database.Connect()
	defer database.Close()

	db := database.GetDB()
	err = database.Migrate(db, migrations.FS, ".")
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migrated successfully")

	// Initialize repositories
	organizerRepo := repository.NewOrganizerRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	eventRepo := repository.NewEventRepository(db)

	// Initialize services
	organizerService := services.NewOrganizerService(organizerRepo)
	venueService := services.NewVenueService(venueRepo, organizerRepo)
	eventService := services.NewEventService(eventRepo, organizerRepo, venueRepo)

	// Initialize handlers
	organizerHandler := handlers.NewOrganizerHandler(organizerService)
	venueHandler := handlers.NewVenueHandler(venueService)
	eventHandler := handlers.NewEventHandler(eventService)

	// Setup Router
	r := router.SetupRouter(organizerHandler, venueHandler, eventHandler)

	// Start server
	port := config.App.HttpServer.Port
	if port == 0 {
		port = 8081
	}
	address := fmt.Sprintf(":%d", port)
	log.Printf("Event server starting on %s", address)
	if err := r.Run(address); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
