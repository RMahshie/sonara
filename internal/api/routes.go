package api

import (
	"database/sql"
	"net/http"

	"github.com/RMahshie/sonara/internal/api/handlers"
	"github.com/RMahshie/sonara/internal/repository"
	"github.com/RMahshie/sonara/internal/storage"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes sets up all API routes
func RegisterRoutes(router *chi.Mux, api huma.API, db *sql.DB, s3Service storage.S3Service, analysisRepo repository.AnalysisRepository) {
	// Initialize handlers
	analysisHandler := handlers.NewAnalysisHandler(analysisRepo, s3Service)

	// Register analysis routes
	huma.Register(api, huma.Operation{
		OperationID: "createAnalysis",
		Method:      http.MethodPost,
		Path:        "/api/analyses",
		Summary:     "Create a new analysis",
		Description: "Creates a new analysis record and returns an upload URL",
		Tags:        []string{"Analysis"},
	}, analysisHandler.CreateAnalysis)

	huma.Register(api, huma.Operation{
		OperationID: "getAnalysisStatus",
		Method:      http.MethodGet,
		Path:        "/api/analyses/{id}/status",
		Summary:     "Get analysis status",
		Description: "Returns the current status and progress of an analysis",
		Tags:        []string{"Analysis"},
	}, analysisHandler.GetAnalysisStatus)

	huma.Register(api, huma.Operation{
		OperationID: "getAnalysisResults",
		Method:      http.MethodGet,
		Path:        "/api/analyses/{id}/results",
		Summary:     "Get analysis results",
		Description: "Returns the complete analysis results including frequency data",
		Tags:        []string{"Analysis"},
	}, analysisHandler.GetAnalysisResults)

	huma.Register(api, huma.Operation{
		OperationID: "addRoomInfo",
		Method:      http.MethodPost,
		Path:        "/api/analyses/{id}/room-info",
		Summary:     "Add room information",
		Description: "Adds room configuration information to an analysis",
		Tags:        []string{"Analysis"},
	}, analysisHandler.AddRoomInfo)
}
