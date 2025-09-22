package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/RMahshie/sonara/internal/repository"
	"github.com/RMahshie/sonara/internal/storage"
	"github.com/RMahshie/sonara/pkg/models"
	"github.com/google/uuid"
)

type ProcessingService interface {
	ProcessAnalysis(ctx context.Context, analysisID uuid.UUID) error
}

type processingService struct {
	s3         storage.S3Service
	repository repository.AnalysisRepository
	pythonPath string // "scripts/analyze_audio.py"
}

func NewProcessingService(s3Service storage.S3Service, repo repository.AnalysisRepository, pythonPath string) ProcessingService {
	return &processingService{
		s3:         s3Service,
		repository: repo,
		pythonPath: pythonPath,
	}
}

func (s *processingService) ProcessAnalysis(ctx context.Context, analysisID uuid.UUID) error {
	// Step 1: Update to processing status
	if err := s.repository.UpdateStatus(ctx, analysisID, "processing", 10); err != nil {
		return err
	}

	// Step 2: Get analysis details
	analysis, err := s.repository.GetByID(ctx, analysisID)
	if err != nil {
		return err
	}

	// Step 3: Download from S3
	if err := s.repository.UpdateStatus(ctx, analysisID, "processing", 20); err != nil {
		return err
	}

	audioData, err := s.s3.DownloadFile(ctx, *analysis.AudioS3Key)
	if err != nil {
		s.repository.UpdateError(ctx, analysisID, "Failed to download audio")
		return err
	}

	// Step 4: Save to temp file
	tempFile := filepath.Join("/tmp", fmt.Sprintf("%s.audio", analysisID))
	if err := os.WriteFile(tempFile, audioData, 0644); err != nil {
		return err
	}
	defer os.Remove(tempFile) // Always cleanup

	// Step 5: Run Python analysis with room data if available
	if err := s.repository.UpdateStatus(ctx, analysisID, "processing", 50); err != nil {
		return err
	}

	// Get room info for enhanced analysis
	roomInfo, err := s.repository.GetRoomInfo(ctx, analysisID)

	var cmd *exec.Cmd
	if err == nil && roomInfo != nil && roomInfo.RoomLength > 0 && roomInfo.RoomWidth > 0 && roomInfo.RoomHeight > 0 {
		// Pass room data to Python for enhanced analysis
		roomData := map[string]interface{}{
			"room_length":                      roomInfo.RoomLength,
			"room_width":                       roomInfo.RoomWidth,
			"room_height":                      roomInfo.RoomHeight,
			"speaker_distance_from_front_wall": roomInfo.SpeakerDistanceFromFrontWall,
		}

		roomDataJSON, err := json.Marshal(roomData)
		if err != nil {
			// Fallback to analysis without room data
			cmd = exec.CommandContext(ctx, "python3", s.pythonPath, tempFile)
		} else {
			cmd = exec.CommandContext(ctx, "python3", s.pythonPath, tempFile, string(roomDataJSON))
		}
	} else {
		// No room data available, use original approach
		cmd = exec.CommandContext(ctx, "python3", s.pythonPath, tempFile)
	}

	output, err := cmd.Output()
	if err != nil {
		s.repository.UpdateError(ctx, analysisID, "Audio analysis failed")
		return fmt.Errorf("python analysis failed: %w", err)
	}

	// Step 6: Parse results
	if err := s.repository.UpdateStatus(ctx, analysisID, "processing", 80); err != nil {
		return err
	}
	var result struct {
		FrequencyData []models.FrequencyPoint `json:"frequency_data"`
		RT60          float64                 `json:"rt60"`
		RoomModes     interface{}             `json:"room_modes"` // Can be []float64 or enhanced format
		Error         string                  `json:"error,omitempty"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("failed to parse results: %w", err)
	}

	if result.Error != "" {
		s.repository.UpdateError(ctx, analysisID, result.Error)
		return fmt.Errorf("analysis error: %s", result.Error)
	}

	// Step 7: Store results
	if err := s.repository.UpdateStatus(ctx, analysisID, "processing", 90); err != nil {
		return err
	}

	// Convert room modes to the expected format
	var roomModes []float64
	if modes, ok := result.RoomModes.([]interface{}); ok {
		// Enhanced format - extract frequencies from mode objects
		for _, mode := range modes {
			if modeMap, ok := mode.(map[string]interface{}); ok {
				if freq, ok := modeMap["frequency"].(float64); ok {
					roomModes = append(roomModes, freq)
				}
			}
		}
	} else if modes, ok := result.RoomModes.([]float64); ok {
		// Original format
		roomModes = modes
	}

	if err := s.repository.StoreResults(ctx, &models.AnalysisResults{
		ID:            uuid.New().String(),
		AnalysisID:    analysis.ID,
		FrequencyData: result.FrequencyData,
		RT60:          &result.RT60,
		RoomModes:     roomModes,
		CreatedAt:     analysis.CreatedAt,
	}); err != nil {
		return err
	}

	// Step 8: Mark complete
	if err := s.repository.UpdateStatus(ctx, analysisID, "completed", 100); err != nil {
		return err
	}

	return nil
}
