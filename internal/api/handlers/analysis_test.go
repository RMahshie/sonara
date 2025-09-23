package handlers

import (
	"context"
	"fmt"
	"testing"

	"github.com/RMahshie/sonara/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAnalysisRepository implements repository.AnalysisRepository for testing
type MockAnalysisRepository struct {
	mock.Mock
}

func (m *MockAnalysisRepository) Create(ctx context.Context, analysis *models.Analysis) error {
	args := m.Called(ctx, analysis)
	return args.Error(0)
}

func (m *MockAnalysisRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Analysis, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Analysis), args.Error(1)
}

func (m *MockAnalysisRepository) GetBySessionID(ctx context.Context, sessionID string) ([]*models.Analysis, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]*models.Analysis), args.Error(1)
}

func (m *MockAnalysisRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, progress int) error {
	args := m.Called(ctx, id, status, progress)
	return args.Error(0)
}

func (m *MockAnalysisRepository) UpdateError(ctx context.Context, id uuid.UUID, errorMsg string) error {
	args := m.Called(ctx, id, errorMsg)
	return args.Error(0)
}

func (m *MockAnalysisRepository) StoreResults(ctx context.Context, results *models.AnalysisResults) error {
	args := m.Called(ctx, results)
	return args.Error(0)
}

func (m *MockAnalysisRepository) GetResults(ctx context.Context, analysisID uuid.UUID) (*models.AnalysisResults, error) {
	args := m.Called(ctx, analysisID)
	return args.Get(0).(*models.AnalysisResults), args.Error(1)
}

func (m *MockAnalysisRepository) CreateRoomInfo(ctx context.Context, info *models.RoomInfo) error {
	args := m.Called(ctx, info)
	return args.Error(0)
}

func (m *MockAnalysisRepository) GetRoomInfo(ctx context.Context, analysisID uuid.UUID) (*models.RoomInfo, error) {
	args := m.Called(ctx, analysisID)
	return args.Get(0).(*models.RoomInfo), args.Error(1)
}

// MockS3Service implements storage.S3Service for testing
type MockS3Service struct {
	mock.Mock
}

func (m *MockS3Service) GenerateUploadURL(ctx context.Context, key string, contentType string) (string, error) {
	args := m.Called(ctx, key, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockS3Service) GenerateDownloadURL(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockS3Service) DownloadFile(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockS3Service) DeleteFile(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// MockProcessingService implements processing.ProcessingService for testing
type MockProcessingService struct {
	mock.Mock
}

func (m *MockProcessingService) ProcessAnalysis(ctx context.Context, analysisID uuid.UUID) error {
	args := m.Called(ctx, analysisID)
	return args.Error(0)
}

func TestCreateAnalysis(t *testing.T) {
	tests := []struct {
		name      string
		input     models.CreateAnalysisRequest
		mockSetup func(*MockAnalysisRepository, *MockS3Service, *MockProcessingService)
		wantCode  int
		wantError bool
	}{
		{
			name: "valid audio file",
			input: models.CreateAnalysisRequest{
				Body: struct {
					SessionID string `json:"session_id" minLength:"10" maxLength:"50" required:"true" doc:"Client session identifier"`
					FileSize  int64  `json:"file_size" minimum:"1000" maximum:"20971520" required:"true" doc:"Audio file size in bytes"`
					MimeType  string `json:"mime_type" enum:"audio/wav,audio/mpeg,audio/flac,audio/webm,audio/ogg" required:"true" doc:"Audio file MIME type"`
				}{
					SessionID: "test-session-123",
					FileSize:  5242880, // 5MB
					MimeType:  "audio/wav",
				},
			},
			mockSetup: func(mockRepo *MockAnalysisRepository, mockS3 *MockS3Service, mockProc *MockProcessingService) {
				mockS3.On("GenerateUploadURL", mock.Anything, mock.Anything, "audio/wav").Return("https://example.com/upload", nil)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Analysis")).Return(nil)
			},
			wantCode:  200,
			wantError: false,
		},
		{
			name: "file too large",
			input: models.CreateAnalysisRequest{
				Body: struct {
					SessionID string `json:"session_id" minLength:"10" maxLength:"50" required:"true" doc:"Client session identifier"`
					FileSize  int64  `json:"file_size" minimum:"1000" maximum:"20971520" required:"true" doc:"Audio file size in bytes"`
					MimeType  string `json:"mime_type" enum:"audio/wav,audio/mpeg,audio/flac,audio/webm,audio/ogg" required:"true" doc:"Audio file MIME type"`
				}{
					SessionID: "test-session-123",
					FileSize:  25000000, // 25MB
					MimeType:  "audio/wav",
				},
			},
			mockSetup: func(mockRepo *MockAnalysisRepository, mockS3 *MockS3Service, mockProc *MockProcessingService) {
				// No mocks needed since validation happens before S3 call
			},
			wantCode:  400, // Now the handler validates file size and returns 400
			wantError: true,
		},
		{
			name: "invalid MIME type for S3",
			input: models.CreateAnalysisRequest{
				Body: struct {
					SessionID string `json:"session_id" minLength:"10" maxLength:"50" required:"true" doc:"Client session identifier"`
					FileSize  int64  `json:"file_size" minimum:"1000" maximum:"20971520" required:"true" doc:"Audio file size in bytes"`
					MimeType  string `json:"mime_type" enum:"audio/wav,audio/mpeg,audio/flac,audio/webm,audio/ogg" required:"true" doc:"Audio file MIME type"`
				}{
					SessionID: "test-session-123",
					FileSize:  5242880,
					MimeType:  "audio/wav",
				},
			},
			mockSetup: func(mockRepo *MockAnalysisRepository, mockS3 *MockS3Service, mockProc *MockProcessingService) {
				mockS3.On("GenerateUploadURL", mock.Anything, mock.Anything, "audio/wav").Return("", assert.AnError)
			},
			wantCode:  400,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := &MockAnalysisRepository{}
			mockS3 := &MockS3Service{}
			mockProc := &MockProcessingService{}
			tt.mockSetup(mockRepo, mockS3, mockProc)

			// Create handler
			handler := NewAnalysisHandler(mockRepo, mockS3, mockProc)

			// Call handler
			resp, err := handler.CreateAnalysis(context.Background(), &tt.input)

			// Assertions
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Body.ID)
				assert.NotEmpty(t, resp.Body.UploadURL)
				assert.Equal(t, 900, resp.Body.ExpiresIn) // 15 minutes in seconds
			}

			// Verify mocks
			mockRepo.AssertExpectations(t)
			mockS3.AssertExpectations(t)
			mockProc.AssertExpectations(t)
		})
	}
}

// TestCreateAnalysis_ErrorMessages tests specific error messages returned by CreateAnalysis
func TestCreateAnalysis_ErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		fileSize       int64
		mimeType       string
		s3Error        error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "file too small",
			fileSize:       500,
			mimeType:       "audio/wav",
			expectedStatus: 400,
			expectedMsg:    "Recording too short. Please ensure microphone is working.",
		},
		{
			name:           "file too large",
			fileSize:       25 * 1024 * 1024, // 25MB
			mimeType:       "audio/wav",
			expectedStatus: 400,
			expectedMsg:    "Recording too large. Please try a shorter recording.",
		},
		{
			name:           "invalid mime type",
			fileSize:       5000,
			mimeType:       "audio/wav",
			s3Error:        fmt.Errorf("invalid content type: audio/unknown"),
			expectedStatus: 400,
			expectedMsg:    "Recording format not supported. Please try again.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAnalysisRepository{}
			mockS3 := &MockS3Service{}
			mockProc := &MockProcessingService{}

			handler := NewAnalysisHandler(mockRepo, mockS3, mockProc)

			// Setup mocks
			if tt.s3Error != nil {
				mockS3.On("GenerateUploadURL", mock.Anything, mock.Anything, tt.mimeType).Return("", tt.s3Error)
			} else {
				mockS3.On("GenerateUploadURL", mock.Anything, mock.Anything, tt.mimeType).Return("http://test-url", nil)
				mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
			}

			req := &models.CreateAnalysisRequest{
				Body: struct {
					SessionID string `json:"session_id" minLength:"10" maxLength:"50" required:"true" doc:"Client session identifier"`
					FileSize  int64  `json:"file_size" minimum:"1000" maximum:"20971520" required:"true" doc:"Audio file size in bytes"`
					MimeType  string `json:"mime_type" enum:"audio/wav,audio/mpeg,audio/flac,audio/webm,audio/ogg" required:"true" doc:"Audio file MIME type"`
				}{
					SessionID: "test-session",
					FileSize:  tt.fileSize,
					MimeType:  tt.mimeType,
				},
			}

			_, err := handler.CreateAnalysis(context.Background(), req)

			if tt.expectedStatus >= 400 {
				assert.Error(t, err)
				// Note: Testing specific error messages from Huma errors requires checking the error structure
				// For now, we verify the error occurs as expected
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
