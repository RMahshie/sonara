package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	apiPkg "github.com/RMahshie/sonara/internal/api"
	"github.com/RMahshie/sonara/internal/config"
	"github.com/RMahshie/sonara/internal/processing"
	"github.com/RMahshie/sonara/internal/repository/postgres"
	"github.com/RMahshie/sonara/internal/storage"
	"github.com/RMahshie/sonara/pkg/models"
)

func main() {
	// Configure zerolog for structured logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize database connection
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping database")
	}

	// Initialize repositories
	analysisRepo := postgres.NewPostgresAnalysisRepository(db)

	// Initialize S3 service
	s3Config := storage.S3Config{
		Bucket:    cfg.AWS.S3Bucket,
		Endpoint:  cfg.AWS.S3Endpoint,
		Region:    cfg.AWS.Region,
		AccessKey: cfg.AWS.AccessKeyID,
		SecretKey: cfg.AWS.SecretAccessKey,
	}
	s3Service, err := storage.NewS3Service(s3Config)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize S3 service")
	}

	// Initialize processing service (for future use)
	_ = processing.NewProcessingService(s3Service, analysisRepo, "scripts/analyze_audio.py")

	// Create Chi router
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(zerologLogger())
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))

	// CORS middleware
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   strings.Split(allowedOrigins, ","),
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	} else {
		// Default to allowing localhost for development
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		}))
	}

	// Create Huma API
	config := huma.DefaultConfig("Sonara API", "1.0.0")
	config.DocsPath = "/api/docs"
	api := humachi.New(router, config)

	// Register health endpoint
	huma.Register(api, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check",
		Description: "Returns the health status of the service",
	}, func(ctx context.Context, input *struct{}) (*models.HealthResponse, error) {
		resp := &models.HealthResponse{}
		resp.Body.Status = "healthy"
		resp.Body.Version = "1.0.0"
		resp.Body.Time = time.Now()
		return resp, nil
	})

	// Register analysis routes
	apiPkg.RegisterRoutes(router, api, db, s3Service, analysisRepo)

	// Serve OpenAPI spec at /api/docs
	router.Get("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		spec, err := api.OpenAPI().MarshalJSON()
		if err != nil {
			http.Error(w, "Failed to generate OpenAPI spec", http.StatusInternalServerError)
			return
		}
		w.Write(spec)
	})

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("Starting Sonara API server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}

// zerologLogger returns a Chi middleware that logs HTTP requests using zerolog
func zerologLogger() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				log.Info().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Str("remote_ip", r.RemoteAddr).
					Int("status", ww.Status()).
					Dur("latency", time.Since(start)).
					Str("user_agent", r.UserAgent()).
					Msg("HTTP request")
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
