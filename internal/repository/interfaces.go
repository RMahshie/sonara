package repository

import (
	"context"

	"github.com/RMahshie/sonara/pkg/models"
	"github.com/google/uuid"
)

// RoomDimensions represents just the room dimensions needed for analysis
type RoomDimensions struct {
	LengthFeet float64
	WidthFeet  float64
	HeightFeet float64
}

// AnalysisRepository defines the interface for analysis data operations
type AnalysisRepository interface {
	Create(ctx context.Context, analysis *models.Analysis) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Analysis, error)
	GetBySessionID(ctx context.Context, sessionID string) ([]*models.Analysis, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, progress int) error
	UpdateError(ctx context.Context, id uuid.UUID, errorMsg string) error
	StoreResults(ctx context.Context, results *models.AnalysisResults) error
	GetResults(ctx context.Context, analysisID uuid.UUID) (*models.AnalysisResults, error)
	CreateRoomInfo(ctx context.Context, info *models.RoomInfo) error
	GetRoomInfo(ctx context.Context, analysisID uuid.UUID) (*models.RoomInfo, error)
	GetRoomDimensions(ctx context.Context, analysisID uuid.UUID) (*RoomDimensions, error)
}

// RoomInfoRepository defines the interface for room information operations
type RoomInfoRepository interface {
	CreateRoomInfo(ctx context.Context, info *models.RoomInfo) error
	GetRoomInfo(ctx context.Context, analysisID uuid.UUID) (*models.RoomInfo, error)
}

// AIInteractionRepository defines the interface for AI interaction operations
type AIInteractionRepository interface {
	CreateAIInteraction(ctx context.Context, interaction *models.AIInteraction) error
	GetAIInteraction(ctx context.Context, analysisID *string, questionHash string) (*models.AIInteraction, error)
}
