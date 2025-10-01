package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RMahshie/sonara/internal/config"
	"github.com/RMahshie/sonara/internal/repository"
	"github.com/RMahshie/sonara/internal/storage"
	"github.com/RMahshie/sonara/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type ProcessingService interface {
	ProcessAnalysis(ctx context.Context, analysisID uuid.UUID) error
}

type processingService struct {
	s3         storage.S3Service
	repository repository.AnalysisRepository
	pythonPath string   // Absolute path to "scripts/analyze_audio.py"
	pythonArgs []string // Python command arguments (for Docker or direct execution)
}

func NewProcessingService(s3Service storage.S3Service, repo repository.AnalysisRepository, cfg *config.Config, scriptPath string) ProcessingService {
	// Parse PYTHON_CMD from config into command arguments
	pythonArgs := parsePythonCommand(cfg.Processing.PythonCmd)

	// For containerized execution (contains "docker exec"), script path is already included
	// For direct execution ("python3"), append the script path
	if !strings.Contains(cfg.Processing.PythonCmd, "docker exec") {
		pythonArgs = append(pythonArgs, scriptPath)
	}

	return &processingService{
		s3:         s3Service,
		repository: repo,
		pythonPath: scriptPath, // Store the script path for reference
		pythonArgs: pythonArgs,
	}
}

// parsePythonCommand parses a PYTHON_CMD string into command arguments
// Examples:
//
//	"python3" → ["python3"]
//	"docker exec analyzer python /app/analyze_audio.py" → ["docker", "exec", "analyzer", "python", "/app/analyze_audio.py"]
func parsePythonCommand(pythonCmd string) []string {
	if pythonCmd == "" {
		return []string{"python3"} // fallback
	}

	// Split on spaces, but be careful with quoted arguments
	// For now, simple space splitting should work for our use cases
	return strings.Fields(pythonCmd)
}

func (s *processingService) ProcessAnalysis(ctx context.Context, analysisID uuid.UUID) error {
	log.Info().Str("analysisID", analysisID.String()).Msg("Starting audio processing pipeline")

	// Step 1: Update to processing status
	log.Info().Str("analysisID", analysisID.String()).Msg("Step 1: Updating status to processing (10%)")
	if err := s.repository.UpdateStatus(ctx, analysisID, "processing", 10); err != nil {
		return err
	}
	log.Info().Str("analysisID", analysisID.String()).Msg("Status updated to processing successfully")

	// Step 2: Get analysis details
	log.Info().Str("analysisID", analysisID.String()).Msg("Step 2: Retrieving analysis details from database")
	analysis, err := s.repository.GetByID(ctx, analysisID)
	if err != nil {
		return err
	}
	log.Info().Str("analysisID", analysisID.String()).Str("signalID", analysis.SignalID).Str("audioS3Key", *analysis.AudioS3Key).Msg("Analysis details retrieved successfully")

	// Step 3: Download from S3
	log.Info().Str("analysisID", analysisID.String()).Msg("Step 3: Updating status to processing (20%)")
	if err := s.repository.UpdateStatus(ctx, analysisID, "processing", 20); err != nil {
		return err
	}

	// For testing: if S3 key starts with "test-", read from /tmp instead of S3
	// For integration testing: if S3 key starts with "minio-real-", read from /tmp (simulated upload)
	var audioData []byte

	if strings.HasPrefix(*analysis.AudioS3Key, "test-") {
		log.Info().Str("analysisID", analysisID.String()).Str("filePath", "/tmp/"+*analysis.AudioS3Key).Msg("Downloading from test file")
		audioData, err = os.ReadFile("/tmp/" + *analysis.AudioS3Key)
		if err != nil {
			s.repository.UpdateError(ctx, analysisID, "Failed to read test audio file")
			return nil // Don't return error, status is updated to failed
		}
		log.Info().Str("analysisID", analysisID.String()).Int("fileSize", len(audioData)).Msg("Test file downloaded successfully")
	} else if strings.HasPrefix(*analysis.AudioS3Key, "minio-real-") {
		// Integration test: read from simulated MinIO upload location
		log.Info().Str("analysisID", analysisID.String()).Str("filePath", "/tmp/"+*analysis.AudioS3Key).Msg("Downloading from MinIO test file")
		audioData, err = os.ReadFile("/tmp/" + *analysis.AudioS3Key)
		if err != nil {
			s.repository.UpdateError(ctx, analysisID, "Failed to download from MinIO")
			return nil // Don't return error, status is updated to failed
		}
		log.Info().Str("analysisID", analysisID.String()).Int("fileSize", len(audioData)).Msg("MinIO test file downloaded successfully")
	} else {
		log.Info().Str("analysisID", analysisID.String()).Str("s3Key", *analysis.AudioS3Key).Msg("Downloading from S3")
		audioData, err = s.s3.DownloadFile(ctx, *analysis.AudioS3Key)
		if err != nil {
			s.repository.UpdateError(ctx, analysisID, "Failed to download audio")
			return nil // Don't return error, status is updated to failed
		}
		log.Info().Str("analysisID", analysisID.String()).Int("fileSize", len(audioData)).Msg("S3 download completed successfully")
	}

	// Step 4: Save to temp file
	log.Info().Str("analysisID", analysisID.String()).Msg("Step 4: Saving audio data to temp file")
	tempFile := filepath.Join("/tmp", fmt.Sprintf("%s.audio", analysisID))
	if err := os.WriteFile(tempFile, audioData, 0644); err != nil {
		return err
	}
	defer os.Remove(tempFile) // Always cleanup
	log.Info().Str("analysisID", analysisID.String()).Str("tempFile", tempFile).Int("fileSize", len(audioData)).Msg("Temp file created successfully")

	// Step 4.5: Convert WebM/OGG to WAV for Python compatibility
	log.Info().Str("analysisID", analysisID.String()).Msg("Step 4.5: Converting audio to WAV format with FFmpeg")
	wavFile := filepath.Join("/tmp", fmt.Sprintf("%s.wav", analysisID))
	convertCmd := exec.Command("ffmpeg", "-i", tempFile, "-acodec", "pcm_s16le", "-ar", "44100", "-y", wavFile)
	if err := convertCmd.Run(); err != nil {
		return fmt.Errorf("failed to convert audio to WAV: %w", err)
	}
	defer os.Remove(wavFile) // Cleanup WAV file
	log.Info().Str("analysisID", analysisID.String()).Str("wavFile", wavFile).Msg("FFmpeg conversion completed successfully")

	// Step 5: Run Python analysis with sweep deconvolution
	log.Info().Str("analysisID", analysisID.String()).Msg("Step 5: Updating status to processing (50%)")
	if err := s.repository.UpdateStatus(ctx, analysisID, "processing", 50); err != nil {
		return err
	}

	// Get analysis record to obtain signal ID
	log.Info().Str("analysisID", analysisID.String()).Msg("Retrieving updated analysis record for signal ID")
	analysis, err = s.repository.GetByID(ctx, analysisID)
	if err != nil {
		return fmt.Errorf("failed to get analysis: %w", err)
	}

	// Generate unique result file path
	resultFile := filepath.Join("/tmp", fmt.Sprintf("%s.result.json", analysisID))
	defer os.Remove(resultFile) // GUARANTEED cleanup

	// Use configured python command
	log.Info().Str("analysisID", analysisID.String()).Strs("pythonArgs", s.pythonArgs).Str("scriptPath", s.pythonPath).Str("wavFile", wavFile).Str("signalID", analysis.SignalID).Str("resultFile", resultFile).Msg("Starting Python script execution")

	// Pass signal ID and result file path to Python script
	args := append(s.pythonArgs, wavFile, analysis.SignalID, resultFile)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	executionTime := time.Since(startTime)

	if err != nil {
		outputStr := string(output)
		log.Error().Str("analysisID", analysisID.String()).Dur("executionTime", executionTime).Err(err).Str("output", outputStr).Msg("Python script execution failed")
		s.repository.UpdateError(ctx, analysisID, fmt.Sprintf("Audio analysis failed: %s", outputStr))
		return fmt.Errorf("python analysis failed: %w, output: %s", err, outputStr)
	}

	log.Info().Str("analysisID", analysisID.String()).Dur("executionTime", executionTime).Msg("Python script execution completed successfully")

	// Step 6: Parse results
	log.Info().Str("analysisID", analysisID.String()).Msg("Step 6: Updating status to processing (80%)")
	if err := s.repository.UpdateStatus(ctx, analysisID, "processing", 80); err != nil {
		return err
	}

	// Read result from file
	log.Info().Str("analysisID", analysisID.String()).Msg("Reading result file")
	resultData, err := os.ReadFile(resultFile)
	if err != nil {
		log.Error().Str("analysisID", analysisID.String()).Err(err).Msg("Failed to read result file")
		s.repository.UpdateError(ctx, analysisID, "Failed to read analysis results")
		return fmt.Errorf("failed to read result file: %w", err)
	}

	if len(resultData) == 0 {
		log.Error().Str("analysisID", analysisID.String()).Msg("Result file is empty")
		s.repository.UpdateError(ctx, analysisID, "Analysis produced no results")
		return fmt.Errorf("empty result file")
	}

	log.Info().Str("analysisID", analysisID.String()).Int("resultSize", len(resultData)).Msg("Result file read successfully")

	// Parse JSON from file content
	log.Info().Str("analysisID", analysisID.String()).Msg("Parsing JSON results")
	var result struct {
		FrequencyData []models.FrequencyPoint `json:"frequency_data"`
		RT60          float64                 `json:"rt60"`
		RoomModes     interface{}             `json:"room_modes"` // Can be []float64 or enhanced format
		Error         string                  `json:"error,omitempty"`
	}

	if err := json.Unmarshal(resultData, &result); err != nil {
		log.Error().Str("analysisID", analysisID.String()).Err(err).Msg("Failed to parse JSON results")
		return fmt.Errorf("failed to parse results: %w", err)
	}

	if result.Error != "" {
		log.Error().Str("analysisID", analysisID.String()).Str("error", result.Error).Msg("Python script reported error")
		s.repository.UpdateError(ctx, analysisID, result.Error)
		return fmt.Errorf("analysis error: %s", result.Error)
	}

	log.Info().Str("analysisID", analysisID.String()).Int("frequencyPoints", len(result.FrequencyData)).Float64("rt60", result.RT60).Msg("Results parsed successfully")

	// Step 7: Store results
	log.Info().Str("analysisID", analysisID.String()).Msg("Step 7: Updating status to processing (90%)")
	if err := s.repository.UpdateStatus(ctx, analysisID, "processing", 90); err != nil {
		return err
	}

	log.Info().Str("analysisID", analysisID.String()).Msg("Processing and storing analysis results")

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
	log.Info().Str("analysisID", analysisID.String()).Int("roomModes", len(roomModes)).Msg("Storing results in database")

	results := &models.AnalysisResults{
		ID:            uuid.New().String(),
		AnalysisID:    analysis.ID,
		FrequencyData: result.FrequencyData,
		RT60:          &result.RT60,
		RoomModes:     roomModes,
		CreatedAt:     analysis.CreatedAt,
	}

	if err := s.repository.StoreResults(ctx, results); err != nil {
		log.Error().Str("analysisID", analysisID.String()).Err(err).Msg("Failed to store results in database")
		return err
	}

	log.Info().Str("analysisID", analysisID.String()).Str("resultsID", results.ID).Msg("Results stored successfully")

	// Step 8: Mark complete
	log.Info().Str("analysisID", analysisID.String()).Msg("Step 8: Marking analysis as completed (100%)")
	if err := s.repository.UpdateStatus(ctx, analysisID, "completed", 100); err != nil {
		log.Error().Str("analysisID", analysisID.String()).Err(err).Msg("Failed to mark analysis as completed")
		return err
	}

	log.Info().Str("analysisID", analysisID.String()).Msg("Audio processing pipeline completed successfully")
	return nil
}
