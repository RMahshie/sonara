package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RMahshie/sonara/internal/processing"
	"github.com/RMahshie/sonara/internal/repository"
	"github.com/RMahshie/sonara/internal/storage"
	"github.com/RMahshie/sonara/pkg/models"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// AnalysisHandler handles analysis-related HTTP requests
type AnalysisHandler struct {
	repo          repository.AnalysisRepository
	s3Service     storage.S3Service
	processingSvc processing.ProcessingService
}

// NewAnalysisHandler creates a new analysis handler
func NewAnalysisHandler(repo repository.AnalysisRepository, s3Service storage.S3Service, processingSvc processing.ProcessingService) *AnalysisHandler {
	return &AnalysisHandler{
		repo:          repo,
		s3Service:     s3Service,
		processingSvc: processingSvc,
	}
}

// CreateAnalysis creates a new analysis and returns an upload URL
func (h *AnalysisHandler) CreateAnalysis(ctx context.Context, req *models.CreateAnalysisRequest) (*models.CreateAnalysisResponse, error) {
	log.Info().Int64("fileSize", req.Body.FileSize).Str("signalID", req.Body.SignalID).Msg("Creating new analysis")

	// Generate unique analysis ID
	analysisID := uuid.New()
	log.Info().Str("analysisID", analysisID.String()).Msg("Generated analysis ID")

	// Generate S3 key for the audio file
	audioKey := fmt.Sprintf("audio/%s.audio", analysisID)

	// Validate file size explicitly
	if req.Body.FileSize < 1000 {
		return nil, huma.Error400BadRequest("Recording too short. Please ensure microphone is working.", nil)
	}
	if req.Body.FileSize > 20*1024*1024 {
		return nil, huma.Error400BadRequest("Recording too large. Please try a shorter recording.", nil)
	}

	// Generate upload URL
	log.Info().Str("audioKey", audioKey).Str("mimeType", req.Body.MimeType).Msg("Generating S3 upload URL")
	uploadURL, err := h.s3Service.GenerateUploadURL(ctx, audioKey, req.Body.MimeType)
	if err != nil {
		// Check for specific error types and return user-friendly messages
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid content type") {
			return nil, huma.Error400BadRequest("Recording format not supported. Please try again.", err)
		}
		return nil, huma.Error400BadRequest("Failed to prepare upload. Please try again.", err)
	}
	log.Info().Str("uploadURL", uploadURL).Msg("S3 upload URL generated successfully")

	// Create analysis record in database
	analysis := &models.Analysis{
		ID:         analysisID.String(),
		SessionID:  req.Body.SessionID,
		SignalID:   req.Body.SignalID, // Test signal identifier
		Status:     "pending",
		Progress:   0,
		AudioS3Key: &audioKey,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	log.Info().Str("analysisID", analysisID.String()).Str("sessionID", req.Body.SessionID).Msg("Creating analysis record in database")
	if err := h.repo.Create(ctx, analysis); err != nil {
		return nil, huma.Error500InternalServerError("Failed to create analysis", err)
	}
	log.Info().Str("analysisID", analysisID.String()).Msg("Analysis record created successfully")

	log.Info().Str("analysisID", analysisID.String()).Int("expiresIn", int((15 * time.Minute).Seconds())).Msg("Analysis created successfully, returning upload URL to client")
	return &models.CreateAnalysisResponse{
		Body: models.CreateAnalysisResponseBody{
			ID:        analysis.ID,
			UploadURL: uploadURL,
			ExpiresIn: int((15 * time.Minute).Seconds()), // 15 minutes
		},
	}, nil
}

// GetAnalysisStatus returns the current status of an analysis
func (h *AnalysisHandler) GetAnalysisStatus(ctx context.Context, req *models.GetAnalysisStatusRequest) (*models.GetAnalysisStatusResponse, error) {
	log.Info().Str("analysisID", req.ID).Msg("Status check request received")
	analysisID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid analysis ID", err)
	}

	analysis, err := h.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, huma.Error404NotFound("Analysis not found", err)
	}

	// Generate human-readable message based on status and progress
	message := h.generateStatusMessage(analysis.Status, analysis.Progress)

	var resultsID *string
	if analysis.Status == "completed" {
		// Get results ID if completed
		results, err := h.repo.GetResults(ctx, analysisID)
		if err == nil && results != nil {
			resultsID = &results.ID
		}
	}

	log.Info().Str("analysisID", analysis.ID).Str("status", analysis.Status).Int("progress", analysis.Progress).Msg("Returning analysis status")
	return &models.GetAnalysisStatusResponse{
		Body: models.GetAnalysisStatusResponseBody{
			ID:        analysis.ID,
			Status:    analysis.Status,
			Progress:  analysis.Progress,
			Message:   message,
			ResultsID: resultsID,
		},
	}, nil
}

// GetAnalysisResults returns the analysis results
func (h *AnalysisHandler) GetAnalysisResults(ctx context.Context, req *models.GetAnalysisResultsRequest) (*models.GetAnalysisResultsResponse, error) {
	analysisID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid analysis ID", err)
	}

	// Get analysis to verify it exists and is completed
	analysis, err := h.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, huma.Error404NotFound("Analysis not found", err)
	}

	if analysis.Status != "completed" {
		return nil, huma.Error409Conflict("Analysis not yet completed",
			fmt.Errorf("analysis status is %s", analysis.Status))
	}

	// Get results
	results, err := h.repo.GetResults(ctx, analysisID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get results", err)
	}

	// Get room info if available
	roomInfo, _ := h.repo.GetRoomInfo(ctx, analysisID) // Ignore error if no room info

	return &models.GetAnalysisResultsResponse{
		Body: models.GetAnalysisResultsResponseBody{
			ID:            results.ID,
			FrequencyData: results.FrequencyData,
			RT60:          results.RT60,
			RoomModes:     results.RoomModes,
			RoomInfo:      roomInfo,
			CreatedAt:     results.CreatedAt,
		},
	}, nil
}

// StartProcessing starts processing an uploaded file
func (h *AnalysisHandler) StartProcessing(ctx context.Context, req *models.StartProcessingRequest) (*models.StartProcessingResponse, error) {
	log.Info().Str("analysisID", req.ID).Msg("Processing start request received")
	analysisID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid analysis ID", err)
	}

	// Verify analysis exists
	log.Info().Str("analysisID", analysisID.String()).Msg("Verifying analysis exists")
	_, err = h.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, huma.Error404NotFound("Analysis not found", err)
	}
	log.Info().Str("analysisID", analysisID.String()).Msg("Analysis verification successful")

	// Start processing in background (don't wait for completion)
	log.Info().Str("analysisID", analysisID.String()).Msg("Starting background processing goroutine")
	go func() {
		err := h.processingSvc.ProcessAnalysis(context.Background(), analysisID)
		if err != nil {
			// Update status to failed
			h.repo.UpdateError(context.Background(), analysisID, fmt.Sprintf("Processing failed: %v", err))
		}
	}()

	return &models.StartProcessingResponse{
		Body: struct {
			Message string `json:"message" doc:"Confirmation message"`
		}{
			Message: "Processing started successfully",
		},
	}, nil
}

// AddRoomInfo adds room information to an analysis
func (h *AnalysisHandler) AddRoomInfo(ctx context.Context, req *models.AddRoomInfoRequest) (*models.AddRoomInfoResponse, error) {
	analysisID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid analysis ID", err)
	}

	// Verify analysis exists
	_, err = h.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, huma.Error404NotFound("Analysis not found", err)
	}

	// Create room info record
	roomInfo := &models.RoomInfo{
		ID:             uuid.New().String(),
		AnalysisID:     analysisID.String(),
		RoomLengthFeet: req.Body.RoomLengthFeet,
		RoomWidthFeet:  req.Body.RoomWidthFeet,
		RoomHeightFeet: req.Body.RoomHeightFeet,
		CreatedAt:      time.Now(),
	}

	if err := h.repo.CreateRoomInfo(ctx, roomInfo); err != nil {
		return nil, huma.Error500InternalServerError("Failed to save room info", err)
	}

	return &models.AddRoomInfoResponse{
		Body: roomInfo,
	}, nil
}

// generateStatusMessage creates a human-readable status message
func (h *AnalysisHandler) generateStatusMessage(status string, progress int) string {
	switch status {
	case "pending":
		return "Analysis queued for processing..."
	case "processing":
		if progress < 25 {
			return "Starting analysis..."
		} else if progress < 50 {
			return "Downloading audio file..."
		} else if progress < 75 {
			return "Analyzing frequency response..."
		} else {
			return "Finalizing results..."
		}
	case "completed":
		return "Analysis complete!"
	case "failed":
		return "Analysis failed. Please try again."
	default:
		return "Unknown status"
	}
}
