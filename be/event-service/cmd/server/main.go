package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/TranVinhHien/ticket-date/be/common/grpc/interceptor"
	pbEvent "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/event"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/config"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/database"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/database/migrations"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/repository"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/router"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/services"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"time"
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

	// Start gRPC Server
	grpcAddr := fmt.Sprintf(":%d", config.App.HttpServer.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Printf("Warning: failed to listen on gRPC port %d: %v", config.App.HttpServer.GRPCPort, err)
	} else {
		grpcServer := grpc.NewServer(
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             5 * time.Second,
				PermitWithoutStream: true,
			}),
			grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle:     15 * time.Minute,
				MaxConnectionAge:      30 * time.Minute,
				MaxConnectionAgeGrace: 5 * time.Minute,
				Time:                  1 * time.Minute,
				Timeout:               20 * time.Second,
			}),
			grpc.ChainUnaryInterceptor(interceptor.DefaultServerUnaryInterceptors(nil)...),
			grpc.ChainStreamInterceptor(interceptor.DefaultServerStreamInterceptors(nil)...),
		)
		pbEvent.RegisterEventServiceServer(grpcServer, handlers.NewEventGRPCHandler(eventService))

		go func() {
			log.Printf("Event gRPC server starting on %s", grpcAddr)
			if err := grpcServer.Serve(lis); err != nil {
				log.Printf("Event gRPC server stopped: %v", err)
			}
		}()
		defer grpcServer.GracefulStop()
	}

	// Setup Router
	r := router.SetupRouter(organizerHandler, venueHandler, eventHandler)

	// Start server
	address := fmt.Sprintf(":%d", config.App.HttpServer.Port)
	log.Printf("Event server starting on %s", address)
	if err := r.Run(address); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
