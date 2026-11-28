package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TranVinhHien/ticket-date/be/common/cache"
	"github.com/TranVinhHien/ticket-date/be/common/grpc/interceptor"
	pbInventory "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/inventory"
	commonKafka "github.com/TranVinhHien/ticket-date/be/common/kafka"
	"github.com/TranVinhHien/ticket-date/be/common/kafka/events"
	_ "github.com/TranVinhHien/ticket-date/be/inventory-service/docs"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/config"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/consumers"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/cron"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database/migrations"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/repository"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/router"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/services"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
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
	inventoryRepo := repository.NewInventoryRepository(pool)

	// Initialize services
	inventoryService := services.NewInventoryService(inventoryRepo, cacheEngine)

	// Initialize and start Kafka consumer if brokers and group_id are configured
	if len(config.App.Kafka.Brokers) > 0 && config.App.Kafka.GroupID != "" {
		zlog := zerolog.New(os.Stdout).With().Timestamp().Str("service", config.App.App.Name).Logger()
		kafkaConsumer, err := commonKafka.NewConsumer(commonKafka.ConsumerConfig{
			Brokers:      config.App.Kafka.Brokers,
			ClientID:     config.App.Kafka.ClientID,
			GroupID:      config.App.Kafka.GroupID,
			Topics:       []string{events.TopicOrderEvents},
			MaxRetries:   config.App.Kafka.MaxRetries,
			RetryBackoff: config.App.Kafka.RetryBackoff,
			DLQEnabled:   config.App.Kafka.DLQEnabled,
			DLQSuffix:    config.App.Kafka.DLQSuffix,
		}, zlog)
		if err != nil {
			log.Printf("Warning: failed to initialize Kafka consumer: %v", err)
		} else {
			consumers.RegisterOrderEventHandlers(kafkaConsumer, inventoryService)
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

	// Initialize and start cron jobs
	if config.App.Cron.Enabled {
		cronManager, err := cron.NewConfiguredManager(
			config.App.Cron,
			inventoryRepo,
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
	inventoryHandler := handlers.NewInventoryHandler(inventoryService)

	// Start gRPC Server
	grpcAddr := fmt.Sprintf(":%d", config.App.HttpServer.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Printf("Warning: failed to listen on gRPC port %d: %v", config.App.HttpServer.GRPCPort, err)
	} else {
		grpcServer := grpc.NewServer(
			grpc.MaxConcurrentStreams(2048),
			grpc.InitialWindowSize(1<<20),
			grpc.InitialConnWindowSize(1<<21),
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             10 * time.Second,
				PermitWithoutStream: true,
			}),
			grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle:     30 * time.Minute,
				MaxConnectionAge:      2 * time.Hour,
				MaxConnectionAgeGrace: 5 * time.Minute,
				Time:                  2 * time.Minute,
				Timeout:               20 * time.Second,
			}),
			grpc.ChainUnaryInterceptor(interceptor.DefaultServerUnaryInterceptors(nil)...),
			grpc.ChainStreamInterceptor(interceptor.DefaultServerStreamInterceptors(nil)...),
		)
		pbInventory.RegisterInventoryServiceServer(grpcServer, handlers.NewInventoryGRPCHandler(inventoryService))

		go func() {
			log.Printf("gRPC server starting on %s", grpcAddr)
			if err := grpcServer.Serve(lis); err != nil {
				log.Printf("gRPC server stopped: %v", err)
			}
		}()
		defer grpcServer.GracefulStop()
	}

	// Setup Router
	r := router.SetupRouter(inventoryHandler)

	// Start server
	address := fmt.Sprintf(":%d", config.App.HttpServer.Port)
	log.Printf("Server starting on %s", address)
	if err := r.Run(address); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
