//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/TranVinhHien/ticket-date/be/event-service/internal/config"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/database"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/database/migrations"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/repository"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/router"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDB *gorm.DB
var testRouter *gin.Engine

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	var dsn string
	var container testcontainers.Container
	var err error

	if envDSN := os.Getenv("TEST_DB_DSN"); envDSN != "" {
		dsn = envDSN
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		pgContainer, err := tcpostgres.Run(ctx, "postgres:16-alpine",
			tcpostgres.WithDatabase("ticket"),
			tcpostgres.WithUsername("test"),
			tcpostgres.WithPassword("test"),
			tcpostgres.BasicWaitStrategies(),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
			os.Exit(1)
		}
		container = pgContainer

		dsn, err = pgContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			fmt.Fprintf(os.Stderr, "postgres connection string: %v\n", err)
			os.Exit(1)
		}
	}

	exitCode := 1
	defer func() {
		if testDB != nil {
			if sqlDB, dbErr := testDB.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
		}
		if container != nil {
			if err := testcontainers.TerminateContainer(container); err != nil {
				fmt.Fprintf(os.Stderr, "terminate postgres container: %v\n", err)
			}
		}
		os.Exit(exitCode)
	}()

	testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "open postgres: %v\n", err)
		return
	}

	if err = database.Migrate(testDB, migrations.FS, "."); err != nil {
		fmt.Fprintf(os.Stderr, "migrate postgres: %v\n", err)
		return
	}

	config.App.Token.Secret = "integration-secret"

	organizerRepo := repository.NewOrganizerRepository(testDB)
	venueRepo := repository.NewVenueRepository(testDB)
	eventRepo := repository.NewEventRepository(testDB)

	organizerService := services.NewOrganizerService(organizerRepo)
	venueService := services.NewVenueService(venueRepo, organizerRepo)
	eventService := services.NewEventService(eventRepo, organizerRepo, venueRepo)

	organizerHandler := handlers.NewOrganizerHandler(organizerService)
	venueHandler := handlers.NewVenueHandler(venueService)
	eventHandler := handlers.NewEventHandler(eventService)

	testRouter = router.SetupRouter(organizerHandler, venueHandler, eventHandler)
	exitCode = m.Run()
}
