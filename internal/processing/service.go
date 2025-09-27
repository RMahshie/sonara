package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	pythonPath string // Absolute path to "scripts/analyze_audio.py"
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

	// For testing: if S3 key starts with "test-", read from /tmp instead of S3
	// For integration testing: if S3 key starts with "minio-real-", read from /tmp (simulated upload)
	var audioData []byte

	if strings.HasPrefix(*analysis.AudioS3Key, "test-") {
		audioData, err = os.ReadFile("/tmp/" + *analysis.AudioS3Key)
		if err != nil {
			s.repository.UpdateError(ctx, analysisID, "Failed to read test audio file")
			return nil // Don't return error, status is updated to failed
		}
	} else if strings.HasPrefix(*analysis.AudioS3Key, "minio-real-") {
		// Integration test: read from simulated MinIO upload location
		audioData, err = os.ReadFile("/tmp/" + *analysis.AudioS3Key)
		if err != nil {
			s.repository.UpdateError(ctx, analysisID, "Failed to download from MinIO")
			return nil // Don't return error, status is updated to failed
		}
	} else {
		audioData, err = s.s3.DownloadFile(ctx, *analysis.AudioS3Key)
		if err != nil {
			s.repository.UpdateError(ctx, analysisID, "Failed to download audio")
			return nil // Don't return error, status is updated to failed
		}
	}

	// Step 4: Save to temp file
	tempFile := filepath.Join("/tmp", fmt.Sprintf("%s.audio", analysisID))
	if err := os.WriteFile(tempFile, audioData, 0644); err != nil {
		return err
	}
	defer os.Remove(tempFile) // Always cleanup

	// Step 4.5: Convert WebM/OGG to WAV for Python compatibility
	wavFile := filepath.Join("/tmp", fmt.Sprintf("%s.wav", analysisID))
	convertCmd := exec.Command("ffmpeg", "-i", tempFile, "-acodec", "pcm_s16le", "-ar", "48000", "-y", wavFile)
	if err := convertCmd.Run(); err != nil {
		return fmt.Errorf("failed to convert audio to WAV: %w", err)
	}
	defer os.Remove(wavFile) // Cleanup WAV file

	// Step 5: Run Python analysis with room data if available
	if err := s.repository.UpdateStatus(ctx, analysisID, "processing", 50); err != nil {
		return err
	}

	// Get room info for enhanced analysis
	roomInfo, err := s.repository.GetRoomInfo(ctx, analysisID)

	// Use virtual environment python
	pythonCmd := "/Users/rmahshie/Downloads/projects/sonara/scripts/venv/bin/python3"

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
			cmd = exec.CommandContext(ctx, pythonCmd, s.pythonPath, wavFile)
		} else {
			cmd = exec.CommandContext(ctx, pythonCmd, s.pythonPath, wavFile, string(roomDataJSON))
		}
	} else {
		// No room data available, use original approach
		cmd = exec.CommandContext(ctx, pythonCmd, s.pythonPath, wavFile)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		fmt.Printf("Python script output: %s\n", outputStr)
		s.repository.UpdateError(ctx, analysisID, fmt.Sprintf("Audio analysis failed: %s", outputStr))
		return fmt.Errorf("python analysis failed: %w, output: %s", err, outputStr)
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
		// Check if it's an array of objects (enhanced format)
		if len(modes) > 0 {
			if _, ok := modes[0].(map[string]interface{}); ok {
				// Enhanced format - extract frequencies from mode objects
				for _, mode := range modes {
					if modeObj, ok := mode.(map[string]interface{}); ok {
						if freq, ok := modeObj["frequency"].(float64); ok {
							roomModes = append(roomModes, freq)
						}
					}
				}
			} else {
				// Simple array of numbers
				for _, mode := range modes {
					if freq, ok := mode.(float64); ok {
						roomModes = append(roomModes, freq)
					}
				}
			}
		}
	} else if modes, ok := result.RoomModes.([]float64); ok {
		// Original format
		roomModes = modes
	}

	// Store results in database

	results := &models.AnalysisResults{
		ID:            uuid.New().String(),
		AnalysisID:    analysis.ID,
		FrequencyData: result.FrequencyData,
		RT60:          &result.RT60,
		RoomModes:     roomModes,
		CreatedAt:     analysis.CreatedAt,
	}

	if err := s.repository.StoreResults(ctx, results); err != nil {
		return err
	}

	// Step 8: Mark complete
	if err := s.repository.UpdateStatus(ctx, analysisID, "completed", 100); err != nil {
		return err
	}

	return nil
}
