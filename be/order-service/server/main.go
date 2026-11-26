package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcClient "github.com/TranVinhHien/ticket-date/be/common/grpc/client"
	pbEvent "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/event"
	pbInventory "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/inventory"
	commonKafka "github.com/TranVinhHien/ticket-date/be/common/kafka"
	"github.com/TranVinhHien/ticket-date/be/common/kafka/events"
	_ "github.com/TranVinhHien/ticket-date/be/order-service/docs"
	"github.com/rs/zerolog"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/cache"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/config"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/consumers"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/cron"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/database"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/database/migrations"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/handlers"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/repository"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/router"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/services"
)

var confPath = flag.String("config", "", "Path to order service config yaml file")

// @title Order Service API
// @version 1.0
// @description Order Service API for ticket date system.
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

	// Connect to Redis
	database.ConnectRedis()
	defer database.CloseRedis()

	pool := database.GetPool()
	err = database.Migrate(pool, migrations.FS, ".")
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	fmt.Println("Database migrated successfully")

	// Initialize cache (uses Redis by default, fallback to in-memory if Redis client is not available)
	var cacheEngine cache.Cache
	if rdb := database.GetRedisClient(); rdb != nil {
		cacheEngine = cache.NewRedisCache(rdb, 5*time.Minute)
	} else {
		cacheEngine = cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	}

	// Initialize repositories
	orderRepo := repository.NewOrderRepository(pool)

	// Initialize gRPC client for inventory service
	var inventoryClient pbInventory.InventoryServiceClient
	if config.App.GRPCClient.InventoryHost != "" && config.App.GRPCClient.InventoryPort != 0 {
		inventoryAddr := fmt.Sprintf("%s:%d", config.App.GRPCClient.InventoryHost, config.App.GRPCClient.InventoryPort)
		conn, err := grpcClient.NewClientConn(
			inventoryAddr,
			grpcClient.WithTimeout(5*time.Second),
			grpcClient.WithInsecure(),
		)
		if err != nil {
			log.Printf("Warning: failed to connect to inventory gRPC server at %s: %v", inventoryAddr, err)
		} else {
			defer conn.Close()
			inventoryClient = pbInventory.NewInventoryServiceClient(conn)
		}
	}

	// Initialize gRPC client for event service
	var eventClient pbEvent.EventServiceClient
	if config.App.GRPCClient.EventHost != "" && config.App.GRPCClient.EventPort != 0 {
		eventAddr := fmt.Sprintf("%s:%d", config.App.GRPCClient.EventHost, config.App.GRPCClient.EventPort)
		eventConn, err := grpcClient.NewClientConn(
			eventAddr,
			grpcClient.WithTimeout(5*time.Second),
			grpcClient.WithInsecure(),
		)
		if err != nil {
			log.Printf("Warning: failed to connect to event gRPC server at %s: %v", eventAddr, err)
		} else {
			defer eventConn.Close()
			eventClient = pbEvent.NewEventServiceClient(eventConn)
		}
	}

	// Initialize services
	orderService := services.NewOrderService(orderRepo, cacheEngine, inventoryClient, eventClient)

	// Initialize Kafka producer if brokers are configured
	var kafkaProducer commonKafka.Producer
	if len(config.App.Kafka.Brokers) > 0 {
		zlog := zerolog.New(os.Stdout).With().Timestamp().Str("service", config.App.App.Name).Logger()
		kafkaProducer, err = commonKafka.NewProducer(commonKafka.ProducerConfig{
			Brokers:            config.App.Kafka.Brokers,
			ClientID:           config.App.Kafka.ClientID,
			Compression:        config.App.Kafka.Compression,
			BatchMaxBytes:      config.App.Kafka.BatchMaxBytes,
			LingerMs:           config.App.Kafka.LingerMs,
			MaxBufferedRecords: config.App.Kafka.MaxBufferedRecords,
			AllowAutoTopic:     config.App.Kafka.AllowAutoTopic,
		}, zlog)
		if err != nil {
			log.Printf("Warning: failed to initialize Kafka producer: %v", err)
		} else {
			defer func() {
				closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = kafkaProducer.Close(closeCtx)
			}()
		}
	}

	// Initialize and start Kafka consumer for inventory events
	if len(config.App.Kafka.Brokers) > 0 && config.App.Kafka.GroupID != "" {
		zlog := zerolog.New(os.Stdout).With().Timestamp().Str("service", config.App.App.Name).Logger()
		kafkaConsumer, err := commonKafka.NewConsumer(commonKafka.ConsumerConfig{
			Brokers:      config.App.Kafka.Brokers,
			ClientID:     config.App.Kafka.ClientID,
			GroupID:      config.App.Kafka.GroupID,
			Topics:       []string{events.EventInventoryReserved, events.EventInventoryReservationFailed},
			MaxRetries:   config.App.Kafka.MaxRetries,
			RetryBackoff: config.App.Kafka.RetryBackoff,
			DLQEnabled:   config.App.Kafka.DLQEnabled,
			DLQSuffix:    config.App.Kafka.DLQSuffix,
		}, zlog)
		if err != nil {
			log.Printf("Warning: failed to initialize Kafka consumer: %v", err)
		} else {
			consumers.RegisterInventoryEventHandlers(kafkaConsumer, orderService)
			consumerCtx, cancelConsumer := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancelConsumer()
			if err := kafkaConsumer.Start(consumerCtx); err != nil {
				log.Printf("Warning: failed to start Kafka consumer: %v", err)
			} else {
				defer func() {
					stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					_ = kafkaConsumer.Stop(stopCtx)
				}()
			}
		}
	}

	// Initialize and start cron jobs
	if config.App.Cron.Enabled {
		cronManager, err := cron.NewConfiguredManager(
			config.App.Cron,
			orderRepo,
			kafkaProducer,
		)
		if err != nil {
			log.Fatalf("Failed to initialize cron manager: %v", err)
		}
		cronCtx, cancelCron := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancelCron()
		cronManager.Start(cronCtx)
		defer cronManager.Stop()
	}

	// Initialize handlers
	orderHandler := handlers.NewOrderHandler(orderService)

	// Setup Router
	r := router.SetupRouter(orderHandler)

	// Start server
	address := fmt.Sprintf(":%d", config.App.HttpServer.Port)
	log.Printf("Server starting on %s", address)
	if err := r.Run(address); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
