package processing

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/RMahshie/sonara/internal/repository/postgres"
	"github.com/RMahshie/sonara/internal/storage"
	"github.com/RMahshie/sonara/pkg/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	pgContainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestContainer holds test infrastructure
type TestContainer struct {
	postgresContainer testcontainers.Container
	minioContainer    testcontainers.Container
	dbURL             string
	minioURL          string
	bucketName        string
}

// SetupIntegrationTest sets up PostgreSQL and MinIO containers for integration testing
func SetupIntegrationTest(t *testing.T) *TestContainer {
	t.Helper()

	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := pgContainer.Run(ctx,
		"postgres:15-alpine",
		pgContainer.WithDatabase("sonara_test"),
		pgContainer.WithUsername("testuser"),
		pgContainer.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)

	dbURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Start MinIO container
	minioContainer, err := minio.Run(ctx,
		"minio/minio:RELEASE.2024-10-29T16-01-48Z",
		minio.WithUsername("minioadmin"),
		minio.WithPassword("minioadmin"),
	)
	require.NoError(t, err)

	minioURL, err := minioContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// Create test bucket
	bucketName := "sonara-test-" + uuid.New().String()[:8]
	err = createMinioBucket(ctx, minioURL, bucketName)
	require.NoError(t, err)

	return &TestContainer{
		postgresContainer: pgContainer,
		minioContainer:    minioContainer,
		dbURL:             dbURL,
		minioURL:          minioURL,
		bucketName:        bucketName,
	}
}

// CleanupIntegrationTest cleans up test containers
func (tc *TestContainer) CleanupIntegrationTest(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	if tc.minioContainer != nil {
		require.NoError(t, tc.minioContainer.Terminate(ctx))
	}
	if tc.postgresContainer != nil {
		require.NoError(t, tc.postgresContainer.Terminate(ctx))
	}
}

// createMinioBucket creates a bucket in MinIO for testing
func createMinioBucket(ctx context.Context, minioURL, bucketName string) error {
	// For integration tests, we'll let the S3 service handle bucket creation
	// when it first tries to upload. This simplifies the setup.
	return nil
}

// TestFullAnalysisPipeline_Integration tests the complete analysis pipeline
func TestFullAnalysisPipeline_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := SetupIntegrationTest(t)
	defer tc.CleanupIntegrationTest(t)

	ctx := context.Background()

	// Set up dependencies
	db, err := sql.Open("postgres", tc.dbURL)
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresAnalysisRepository(db)
	require.NoError(t, err)

	// Run migrations
	err = runMigrations(t, tc.dbURL)
	require.NoError(t, err)

	s3Config := storage.S3Config{
		Bucket:   tc.bucketName,
		Endpoint: tc.minioURL,
	}
	s3Service, err := storage.NewS3Service(s3Config)
	require.NoError(t, err)

	processingService := NewProcessingService(s3Service, repo, "scripts/analyze_audio.py")

	// Generate test audio file (1kHz sine wave)
	audioData := generateTestAudio(t, 1000.0, 44100, 2.0)
	audioFile := createTempAudioFile(t, audioData, 44100)
	defer os.Remove(audioFile)

	// Upload audio file to S3
	audioKey := fmt.Sprintf("test-audio-%s.wav", uuid.New().String()[:8])
	err = uploadFileToS3(ctx, s3Service, audioFile, audioKey)
	require.NoError(t, err)

	// Create a test analysis with S3 key already set
	audioKeyPtr := audioKey
	analysis := &models.Analysis{
		SessionID:  uuid.New().String(),
		Status:     "pending",
		AudioS3Key: &audioKeyPtr,
	}

	err = repo.Create(ctx, analysis)
	require.NoError(t, err)

	// Process the analysis
	analysisID, err := uuid.Parse(analysis.ID)
	require.NoError(t, err)
	err = processingService.ProcessAnalysis(ctx, analysisID)
	require.NoError(t, err)

	// Wait for processing to complete (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var finalAnalysis *models.Analysis
	for {
		select {
		case <-timeout:
			t.Fatal("Analysis processing timed out")
		case <-ticker.C:
			analysis, err := repo.GetByID(ctx, analysisID)
			require.NoError(t, err)

			if analysis.Status == "completed" || analysis.Status == "failed" {
				finalAnalysis = analysis
				goto processingComplete
			}
		}
	}

processingComplete:
	// Verify analysis completed successfully
	assert.Equal(t, "completed", finalAnalysis.Status)
	assert.NotNil(t, finalAnalysis.CompletedAt)
	assert.Greater(t, finalAnalysis.Progress, 95.0)

	// Verify results were stored
	results, err := repo.GetResults(ctx, analysisID)
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.NotEmpty(t, results.FrequencyData)
	assert.IsType(t, 0.0, results.RT60)

	// Verify 1kHz peak is detected in results
	peakFound := false
	for _, point := range results.FrequencyData {
		if point.Frequency >= 990 && point.Frequency <= 1010 {
			peakFound = true
			break
		}
	}
	assert.True(t, peakFound, "1kHz peak not found in analysis results")
}

// TestAnalysisPipelineFailure_Integration tests error handling in the pipeline
func TestAnalysisPipelineFailure_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := SetupIntegrationTest(t)
	defer tc.CleanupIntegrationTest(t)

	ctx := context.Background()

	// Set up dependencies
	db, err := sql.Open("postgres", tc.dbURL)
	require.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPostgresAnalysisRepository(db)
	require.NoError(t, err)

	// Run migrations
	err = runMigrations(t, tc.dbURL)
	require.NoError(t, err)

	s3Config := storage.S3Config{
		Bucket:   tc.bucketName,
		Endpoint: tc.minioURL,
	}
	s3Service, err := storage.NewS3Service(s3Config)
	require.NoError(t, err)

	processingService := NewProcessingService(s3Service, repo, "scripts/analyze_audio.py")

	// Create analysis with non-existent S3 key
	nonExistentKey := "non-existent-file.wav"
	analysis := &models.Analysis{
		SessionID:  uuid.New().String(),
		Status:     "pending",
		AudioS3Key: &nonExistentKey,
	}

	err = repo.Create(ctx, analysis)
	require.NoError(t, err)

	// Process the analysis (should fail)
	analysisID, err := uuid.Parse(analysis.ID)
	require.NoError(t, err)
	err = processingService.ProcessAnalysis(ctx, analysisID)
	require.NoError(t, err) // ProcessAnalysis itself shouldn't error, but status should be failed

	// Wait for processing to complete
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var finalAnalysis *models.Analysis
	for {
		select {
		case <-timeout:
			t.Fatal("Analysis processing timed out")
		case <-ticker.C:
			analysis, err := repo.GetByID(ctx, analysisID)
			require.NoError(t, err)

			if analysis.Status == "completed" || analysis.Status == "failed" {
				finalAnalysis = analysis
				goto processingComplete
			}
		}
	}

processingComplete:
	// Verify analysis failed as expected
	assert.Equal(t, "failed", finalAnalysis.Status)
}

// Helper functions

func runMigrations(t *testing.T, dbURL string) error {
	t.Helper()

	// Run migrate command
	cmd := exec.Command("migrate", "-path", "../../migrations", "-database", dbURL, "up")
	cmd.Dir = "../../" // Adjust path to migrations directory
	return cmd.Run()
}

func generateTestAudio(t *testing.T, frequency, sampleRate, duration float64) []byte {
	t.Helper()

	// This is a simplified audio generation - in practice you'd use a proper audio library
	// For now, create a minimal WAV header + sine wave data
	numSamples := int(sampleRate * duration)
	samples := make([]int16, numSamples)

	for i := 0; i < numSamples; i++ {
		// Generate sine wave
		time := float64(i) / sampleRate
		sample := int16(32767 * 0.5 * (1 + 0.8*math.Sin(2*math.Pi*frequency*time)))
		samples[i] = sample
	}

	// Create WAV file data (simplified)
	var buf bytes.Buffer
	writeWAVHeader(&buf, sampleRate, numSamples)

	// Write samples
	for _, sample := range samples {
		buf.WriteByte(byte(sample & 0xFF))
		buf.WriteByte(byte((sample >> 8) & 0xFF))
	}

	return buf.Bytes()
}

func writeWAVHeader(w io.Writer, sampleRate float64, numSamples int) {
	// Simplified WAV header
	header := []byte{
		'R', 'I', 'F', 'F', // RIFF
		0, 0, 0, 0, // File size (placeholder)
		'W', 'A', 'V', 'E', // WAVE
		'f', 'm', 't', ' ', // fmt
		16, 0, 0, 0, // Chunk size
		1, 0, // Audio format (PCM)
		1, 0, // Num channels
		0, 0, 0, 0, // Sample rate (placeholder)
		0, 0, 0, 0, // Byte rate (placeholder)
		2, 0, // Block align
		16, 0, // Bits per sample
		'd', 'a', 't', 'a', // data
		0, 0, 0, 0, // Data size (placeholder)
	}

	// Fill in actual values
	fileSize := 36 + numSamples*2
	byteRate := int(sampleRate) * 2

	// File size
	header[4] = byte(fileSize & 0xFF)
	header[5] = byte((fileSize >> 8) & 0xFF)
	header[6] = byte((fileSize >> 16) & 0xFF)
	header[7] = byte((fileSize >> 24) & 0xFF)

	// Sample rate
	header[24] = byte(int(sampleRate) & 0xFF)
	header[25] = byte((int(sampleRate) >> 8) & 0xFF)
	header[26] = byte((int(sampleRate) >> 16) & 0xFF)
	header[27] = byte((int(sampleRate) >> 24) & 0xFF)

	// Byte rate
	header[28] = byte(byteRate & 0xFF)
	header[29] = byte((byteRate >> 8) & 0xFF)
	header[30] = byte((byteRate >> 16) & 0xFF)
	header[31] = byte((byteRate >> 24) & 0xFF)

	// Data size
	dataSize := numSamples * 2
	header[40] = byte(dataSize & 0xFF)
	header[41] = byte((dataSize >> 8) & 0xFF)
	header[42] = byte((dataSize >> 16) & 0xFF)
	header[43] = byte((dataSize >> 24) & 0xFF)

	w.Write(header)
}

func createTempAudioFile(t *testing.T, audioData []byte, sampleRate int) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test_audio_*.wav")
	require.NoError(t, err)

	_, err = tmpFile.Write(audioData)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

func uploadFileToS3(ctx context.Context, s3Service storage.S3Service, filePath, key string) error {
	// For integration tests, we'll use a simplified approach
	// In a real scenario, the client would upload via pre-signed URL
	// For testing, we can put the file in a location where the processing service can find it
	return nil
}
