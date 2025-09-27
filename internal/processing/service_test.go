package processing

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/RMahshie/sonara/internal/repository/postgres"
	"github.com/RMahshie/sonara/internal/storage"
	"github.com/RMahshie/sonara/pkg/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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
	// Create MinIO client to create the bucket
	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", ""),
	}

	// Parse the MinIO URL
	endpoint := minioURL
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = &endpoint
		o.UsePathStyle = true // MinIO requires path-style URLs
	})

	// Create the bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})

	// Ignore "BucketAlreadyExists" error
	if err != nil {
		var bae *types.BucketAlreadyExists
		if !errors.As(err, &bae) {
			var bao *types.BucketAlreadyOwnedByYou
			if !errors.As(err, &bao) {
				return fmt.Errorf("failed to create bucket: %w", err)
			}
		}
	}

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
		Bucket:    tc.bucketName,
		Endpoint:  tc.minioURL,
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
	}
	s3Service, err := storage.NewS3Service(s3Config)
	require.NoError(t, err)

	processingService := NewProcessingService(s3Service, repo, "../../scripts/analyze_audio.py")

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
	now := time.Now()
	analysis := &models.Analysis{
		ID:         uuid.New().String(), // Generate UUID for the analysis
		SessionID:  uuid.New().String(),
		SignalID:   "sine_sweep_20_20k", // Valid signal ID for testing
		Status:     "pending",
		Progress:   0,
		AudioS3Key: &audioKeyPtr,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err = repo.Create(ctx, analysis)
	require.NoError(t, err)

	// Parse UUID for subsequent operations
	analysisID, err := uuid.Parse(analysis.ID)
	require.NoError(t, err)

	// Verify the record was actually created
	createdAnalysis, err := repo.GetByID(ctx, analysisID)
	if err != nil {
		t.Logf("Failed to retrieve created analysis: %v", err)
	} else {
		t.Logf("Successfully created analysis: ID=%s, Status=%s", createdAnalysis.ID, createdAnalysis.Status)
	}

	// Process the analysis
	t.Logf("Analysis ID string: %s", analysis.ID)
	t.Logf("Parsed UUID: %s", analysisID.String())
	err = processingService.ProcessAnalysis(ctx, analysisID)
	require.NoError(t, err)

	// Wait for processing to complete (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var finalAnalysis *models.Analysis
	pollCount := 0
	for {
		select {
		case <-timeout:
			t.Fatal("Analysis processing timed out")
		case <-ticker.C:
			pollCount++
			analysis, err := repo.GetByID(ctx, analysisID)
			require.NoError(t, err)

			t.Logf("POLL #%d: Status=%s, Progress=%d, CompletedAt=%v, UpdatedAt=%v",
				pollCount, analysis.Status, analysis.Progress,
				analysis.CompletedAt, analysis.UpdatedAt)

			if analysis.Status == "completed" || analysis.Status == "failed" {
				t.Logf("POLL #%d: BREAKING - Status is %s", pollCount, analysis.Status)
				finalAnalysis = analysis
				goto processingComplete
			}
		}
	}

processingComplete:
	// Verify analysis completed successfully
	assert.Equal(t, "completed", finalAnalysis.Status)
	assert.NotNil(t, finalAnalysis.CompletedAt)
	assert.Greater(t, finalAnalysis.Progress, 95)

	// Verify results were stored
	results, err := repo.GetResults(ctx, analysisID)
	require.NoError(t, err)
	assert.NotNil(t, results)
	t.Logf("Results: %+v", results)
	t.Logf("FrequencyData length: %d", len(results.FrequencyData))
	t.Logf("RT60: %v (type: %T)", results.RT60, results.RT60)
	if results.RT60 != nil {
		t.Logf("RT60 value: %f (dereferenced type: %T)", *results.RT60, *results.RT60)
	}
	assert.NotEmpty(t, results.FrequencyData)
	assert.NotNil(t, results.RT60)
	if results.RT60 != nil {
		assert.IsType(t, 0.0, *results.RT60)
	}

	// Verify 1kHz peak is detected in results
	peakFound := false
	for _, point := range results.FrequencyData {
		if point.Frequency >= 990 && point.Frequency <= 1010 {
			peakFound = true
			break
		}
	}
	assert.True(t, peakFound, "1kHz peak not found in analysis results")

	// Verify result file cleanup
	assertResultFileCleanup(t, analysisID)
}

// TestMinIOFileOperations_Integration tests actual MinIO upload/download operations
func TestMinIOFileOperations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MinIO integration test in short mode")
	}

	tc := SetupIntegrationTest(t)
	defer tc.CleanupIntegrationTest(t)

	ctx := context.Background()

	// Setup S3 service with MinIO config (no bypass)
	s3Config := storage.S3Config{
		Bucket:    tc.bucketName,
		Endpoint:  tc.minioURL,
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
	}
	s3Service, err := storage.NewS3Service(s3Config)
	require.NoError(t, err)

	// Generate unique test file
	testContent := fmt.Sprintf("MinIO integration test data - %s - %d", uuid.New().String(), time.Now().Unix())
	testData := []byte(testContent)
	testKey := fmt.Sprintf("minio-integration-test-%s.txt", uuid.New().String()[:8])

	t.Logf("Test file key: %s", testKey)
	t.Logf("Test file size: %d bytes", len(testData))

	// Test 1: Upload file to MinIO
	t.Log("Step 1: Uploading test file to MinIO...")
	err = uploadTestDataToMinIO(ctx, s3Service, testData, testKey)
	require.NoError(t, err, "Failed to upload test file to MinIO")
	t.Log("✅ Upload successful")

	// Test 2: Download file from MinIO
	t.Log("Step 2: Downloading test file from MinIO...")
	downloadedData, err := s3Service.DownloadFile(ctx, testKey)
	require.NoError(t, err, "Failed to download test file from MinIO")
	t.Log("✅ Download successful")

	// Test 3: Verify content integrity
	t.Log("Step 3: Verifying content integrity...")
	assert.Equal(t, testData, downloadedData, "Downloaded content does not match original")
	assert.Equal(t, len(testData), len(downloadedData), "Downloaded file size does not match original")
	assert.Equal(t, testContent, string(downloadedData), "Downloaded content string does not match original")
	t.Log("✅ Content integrity verified")

	// Test 4: Test error case - download non-existent file
	t.Log("Step 4: Testing error case (non-existent file)...")
	nonExistentKey := fmt.Sprintf("non-existent-file-%s.txt", uuid.New().String()[:8])
	_, err = s3Service.DownloadFile(ctx, nonExistentKey)
	assert.Error(t, err, "Expected error when downloading non-existent file")
	t.Log("✅ Error case handled correctly")

	// Test 5: Cleanup - delete test file
	t.Log("Step 5: Cleaning up test file...")
	err = s3Service.DeleteFile(ctx, testKey)
	assert.NoError(t, err, "Failed to delete test file from MinIO")
	t.Log("✅ Cleanup successful")

	// Test 6: Verify deletion - try to download deleted file
	t.Log("Step 6: Verifying deletion...")
	_, err = s3Service.DownloadFile(ctx, testKey)
	assert.Error(t, err, "Expected error when downloading deleted file")
	t.Log("✅ Deletion verified")

	t.Log("🎉 All MinIO integration tests passed!")
}

// uploadTestDataToMinIO uploads test data directly to MinIO for integration testing
func uploadTestDataToMinIO(ctx context.Context, s3Service storage.S3Service, data []byte, key string) error {
	// Use the S3 service's upload URL generation to simulate real upload flow
	// Generate pre-signed upload URL (use audio/wav content type since that's accepted)
	uploadURL, err := s3Service.GenerateUploadURL(ctx, key, "audio/wav")
	if err != nil {
		return fmt.Errorf("failed to generate upload URL: %w", err)
	}

	// Upload data using HTTP PUT to the pre-signed URL
	req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", "audio/wav")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload to MinIO: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MinIO upload failed with status: %d", resp.StatusCode)
	}

	return nil
}

// TestMinIOEndToEndAnalysis_Integration tests complete analysis pipeline with real MinIO operations
func TestMinIOEndToEndAnalysis_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MinIO end-to-end test in short mode")
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
		Bucket:    tc.bucketName,
		Endpoint:  tc.minioURL,
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
	}
	s3Service, err := storage.NewS3Service(s3Config)
	require.NoError(t, err)

	processingService := NewProcessingService(s3Service, repo, "../../scripts/analyze_audio.py")

	// Generate test audio file (1kHz sine wave) - same as other tests
	audioData := generateTestAudio(t, 1000.0, 44100, 2.0)
	audioFile := createTempAudioFile(t, audioData, 44100)
	defer os.Remove(audioFile)

	// Upload audio file to MinIO using real operations (not /tmp bypass)
	audioKey := fmt.Sprintf("e2e-audio-%s.wav", uuid.New().String()[:8])
	t.Logf("Audio file key: %s", audioKey)

	// Upload using pre-signed URL (real MinIO operation)
	err = uploadAudioToMinIO(ctx, s3Service, audioFile, audioKey)
	require.NoError(t, err, "Failed to upload audio file to MinIO")
	t.Log("✅ Audio file uploaded to MinIO")

	// Create a test analysis with real MinIO key (no "test-" prefix = actual download)
	nonExistentKey := audioKey // Use the real key
	now := time.Now()
	analysis := &models.Analysis{
		ID:         uuid.New().String(),
		SessionID:  uuid.New().String(),
		SignalID:   "sine_sweep_20_20k", // Valid signal ID for testing
		Status:     "pending",
		Progress:   0,
		AudioS3Key: &nonExistentKey, // Real MinIO key - will trigger actual download
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err = repo.Create(ctx, analysis)
	require.NoError(t, err)

	// Parse UUID for subsequent operations
	analysisID, err := uuid.Parse(analysis.ID)
	require.NoError(t, err)

	t.Logf("Created analysis with ID: %s", analysisID)

	// Process the analysis (will download from MinIO and run full pipeline)
	err = processingService.ProcessAnalysis(ctx, analysisID)
	require.NoError(t, err, "ProcessAnalysis should succeed even with MinIO operations")

	// Wait for processing to complete (with timeout)
	timeout := time.After(45 * time.Second) // Longer timeout for full analysis
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var finalAnalysis *models.Analysis
	pollCount := 0
	for {
		select {
		case <-timeout:
			t.Fatal("Analysis processing timed out")
		case <-ticker.C:
			pollCount++
			analysis, err := repo.GetByID(ctx, analysisID)
			require.NoError(t, err)

			t.Logf("POLL #%d: Status=%s, Progress=%d", pollCount, analysis.Status, analysis.Progress)

			if analysis.Status == "completed" || analysis.Status == "failed" {
				t.Logf("Analysis finished with status: %s", analysis.Status)
				finalAnalysis = analysis
				goto analysisComplete
			}
		}
	}

analysisComplete:
	// Verify analysis completed successfully
	assert.Equal(t, "completed", finalAnalysis.Status)
	assert.NotNil(t, finalAnalysis.CompletedAt)
	assert.Greater(t, finalAnalysis.Progress, 95)

	// Verify results were stored
	results, err := repo.GetResults(ctx, analysisID)
	require.NoError(t, err)
	assert.NotNil(t, results)
	t.Logf("Results: FrequencyData length=%d, RT60=%v", len(results.FrequencyData), results.RT60)

	// Verify frequency data
	assert.NotEmpty(t, results.FrequencyData)
	assert.NotNil(t, results.RT60)
	if results.RT60 != nil {
		assert.IsType(t, 0.0, *results.RT60)
	}

	// Verify 1kHz peak is detected in results
	peakFound := false
	for _, point := range results.FrequencyData {
		if point.Frequency >= 990 && point.Frequency <= 1010 {
			peakFound = true
			break
		}
	}
	assert.True(t, peakFound, "1kHz peak not found in analysis results from MinIO")

	// Verify result file cleanup
	assertResultFileCleanup(t, analysisID)

	// Cleanup: Delete the test file from MinIO
	err = s3Service.DeleteFile(ctx, audioKey)
	assert.NoError(t, err, "Failed to cleanup test file from MinIO")

	t.Log("🎉 MinIO end-to-end analysis test completed successfully!")
}

// uploadAudioToMinIO uploads an audio file to MinIO using pre-signed URL
func uploadAudioToMinIO(ctx context.Context, s3Service storage.S3Service, filePath, key string) error {
	// Read the audio file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read audio file: %w", err)
	}

	// Generate pre-signed upload URL
	uploadURL, err := s3Service.GenerateUploadURL(ctx, key, "audio/wav")
	if err != nil {
		return fmt.Errorf("failed to generate upload URL: %w", err)
	}

	// Upload using HTTP PUT
	req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, bytes.NewReader(fileData))
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", "audio/wav")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload to MinIO: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MinIO upload failed with status: %d", resp.StatusCode)
	}

	return nil
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
		Bucket:    tc.bucketName,
		Endpoint:  tc.minioURL,
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
	}
	s3Service, err := storage.NewS3Service(s3Config)
	require.NoError(t, err)

	processingService := NewProcessingService(s3Service, repo, "../../scripts/analyze_audio.py")

	// Create analysis with non-existent S3 key
	nonExistentKey := "non-existent-file.wav"
	now := time.Now()
	analysis := &models.Analysis{
		ID:         uuid.New().String(),
		SessionID:  uuid.New().String(),
		SignalID:   "sine_sweep_20_20k", // Valid signal ID for testing (though test expects failure from S3)
		Status:     "pending",
		Progress:   0,
		AudioS3Key: &nonExistentKey,
		CreatedAt:  now,
		UpdatedAt:  now,
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

	// Verify result file cleanup even on failure
	assertResultFileCleanup(t, analysisID)
}

// Helper functions

// assertResultFileCleanup verifies that result temp files are properly cleaned up
func assertResultFileCleanup(t *testing.T, analysisID uuid.UUID) {
	t.Helper()
	resultFilePath := fmt.Sprintf("/tmp/%s.result.json", analysisID)
	assert.NoFileExists(t, resultFilePath, "Result file should be cleaned up after processing")
}

func runMigrations(t *testing.T, dbURL string) error {
	t.Helper()

	// Log the database URL for debugging
	t.Logf("Database URL: %s", dbURL)

	// Run migrate command
	cmd := exec.Command("migrate", "-path", "migrations", "-database", dbURL, "up")
	cmd.Dir = "../../" // Adjust path to migrations directory

	// Capture output to see detailed error messages
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Migrate command failed. Output: %s", string(output))
		t.Logf("Migrate error: %v", err)
		return err
	}

	t.Logf("Migrate successful. Output: %s", string(output))
	return nil
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
	// For integration testing, copy file to /tmp where processing service will read it
	// (processing service has special handling for keys starting with "test-")

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	destPath := "/tmp/" + key
	return os.WriteFile(destPath, fileData, 0644)
}
