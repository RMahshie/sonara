package models

import (
	"time"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Body struct {
		Status  string    `json:"status" example:"healthy" doc:"Service health status"`
		Version string    `json:"version" example:"1.0.0" doc:"API version"`
		Time    time.Time `json:"time" doc:"Current server time"`
	}
}

// CreateAnalysisRequest represents a request to create a new analysis
type CreateAnalysisRequest struct {
	Body struct {
		SessionID string `json:"session_id" minLength:"10" maxLength:"50" required:"true" doc:"Client session identifier"`
		FileSize  int64  `json:"file_size" minimum:"1000" maximum:"20971520" required:"true" doc:"Audio file size in bytes"`
		MimeType  string `json:"mime_type" enum:"audio/wav,audio/mpeg,audio/flac,audio/webm,audio/ogg" required:"true" doc:"Audio file MIME type"`
		SignalID  string `json:"signal_id" required:"true" doc:"Test signal identifier (e.g., 'sine_sweep_20_20k')"`
	}
}

// CreateAnalysisResponse represents the response from creating an analysis
type CreateAnalysisResponse struct {
	Body struct {
		ID        string `json:"id" doc:"Analysis unique identifier"`
		UploadURL string `json:"upload_url" doc:"Pre-signed S3 URL for file upload"`
		ExpiresIn int    `json:"expires_in" doc:"URL expiration time in seconds"`
	}
}

// CreateAnalysisResponseBody is the body of the create analysis response
type CreateAnalysisResponseBody struct {
	ID        string `json:"id" doc:"Analysis unique identifier"`
	UploadURL string `json:"upload_url" doc:"Pre-signed S3 URL for file upload"`
	ExpiresIn int    `json:"expires_in" doc:"URL expiration time in seconds"`
}

// GetAnalysisStatusRequest represents a request to get analysis status
type GetAnalysisStatusRequest struct {
	ID string `path:"id" doc:"Analysis ID"`
}

// GetAnalysisStatusResponseBody is the body of the status response
type GetAnalysisStatusResponseBody struct {
	ID        string  `json:"id" doc:"Analysis ID"`
	Status    string  `json:"status" enum:"pending,processing,completed,failed" doc:"Analysis status"`
	Progress  int     `json:"progress" minimum:"0" maximum:"100" doc:"Analysis progress percentage"`
	Message   string  `json:"message,omitempty" doc:"Human-readable status message"`
	ResultsID *string `json:"results_id,omitempty" doc:"Results ID when analysis completes"`
}

// GetAnalysisResultsRequest represents a request to get analysis results
type GetAnalysisResultsRequest struct {
	ID string `path:"id" doc:"Analysis ID"`
}

// GetAnalysisResultsResponseBody is the body of the results response
type GetAnalysisResultsResponseBody struct {
	ID            string           `json:"id" doc:"Analysis ID"`
	FrequencyData []FrequencyPoint `json:"frequency_data" doc:"Frequency response data"`
	RT60          *float64         `json:"rt60,omitempty" doc:"Reverberation time in seconds"`
	RoomModes     []float64        `json:"room_modes,omitempty" doc:"Problematic room mode frequencies"`
	RoomInfo      *RoomInfo        `json:"room_info,omitempty" doc:"Room configuration details"`
	CreatedAt     time.Time        `json:"created_at" doc:"Analysis creation timestamp"`
}

// GetAnalysisStatusResponse represents the current status of an analysis
type GetAnalysisStatusResponse struct {
	Body GetAnalysisStatusResponseBody
}

// GetAnalysisResultsResponse represents the complete analysis results
type GetAnalysisResultsResponse struct {
	Body GetAnalysisResultsResponseBody
}

// RoomInfo represents room configuration information
type RoomInfo struct {
	ID               string   `json:"id" doc:"Room info unique identifier"`
	AnalysisID       string   `json:"analysis_id" doc:"Associated analysis ID"`
	RoomSize         string   `json:"room_size" enum:"small,medium,large,very_large" doc:"Room size category"`
	CeilingHeight    string   `json:"ceiling_height" enum:"standard,high,vaulted" doc:"Ceiling height category"`
	FloorType        string   `json:"floor_type" enum:"carpet,hardwood,tile,rug_on_hard" doc:"Floor material type"`
	Features         []string `json:"features" doc:"Room features like windows, curtains, panels"`
	SpeakerPlacement string   `json:"speaker_placement" enum:"desk,stands,shelf,wall" doc:"Speaker placement type"`
	AdditionalNotes  string   `json:"additional_notes" maxLength:"500" doc:"Additional room notes"`

	// Room dimensions for acoustic analysis
	RoomLength float64 `json:"room_length,omitempty" doc:"Room length in meters"`
	RoomWidth  float64 `json:"room_width,omitempty" doc:"Room width in meters"`
	RoomHeight float64 `json:"room_height,omitempty" doc:"Room height in meters"`

	// Speaker positioning
	SpeakerDistanceFromFrontWall float64 `json:"speaker_distance_from_front_wall,omitempty" doc:"Distance from speakers to front wall in meters"`

	CreatedAt time.Time `json:"created_at" doc:"When room info was added"`
}

// AddRoomInfoRequest represents a request to add room information to an analysis
type AddRoomInfoRequest struct {
	ID   string    `path:"id" doc:"Analysis ID"`
	Body *RoomInfo `json:"-"` // Embedded directly in request body
}

// AddRoomInfoResponse represents the response from adding room information
type AddRoomInfoResponse struct {
	Body *RoomInfo `json:"-"` // Return the room info that was saved
}

// AskQuestionRequest represents a request to ask an AI question about analysis results
type AskQuestionRequest struct {
	ID   string `path:"id" doc:"Analysis ID"`
	Body struct {
		Question string `json:"question" minLength:"10" maxLength:"500" required:"true" doc:"Question about the analysis"`
	}
}

// StartProcessingRequest represents a request to start processing an uploaded file
type StartProcessingRequest struct {
	ID string `path:"id" doc:"Analysis ID"`
}

// StartProcessingResponse represents the response from starting processing
type StartProcessingResponse struct {
	Body struct {
		Message string `json:"message" doc:"Confirmation message"`
	}
}

// AskQuestionResponse represents the AI's answer to a question
type AskQuestionResponse struct {
	Body struct {
		Answer string `json:"answer" doc:"AI-generated answer"`
		Cached bool   `json:"cached" doc:"Whether response was cached"`
	}
}

// Analysis represents the core analysis entity (for internal use)
type Analysis struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"session_id"`
	SignalID    string     `json:"signal_id"` // Test signal identifier
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	AudioS3Key  *string    `json:"audio_s3_key,omitempty"`
	ErrorMsg    *string    `json:"error_message,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// AnalysisResults represents the stored analysis results
type AnalysisResults struct {
	ID            string                 `json:"id"`
	AnalysisID    string                 `json:"analysis_id"`
	FrequencyData []FrequencyPoint       `json:"frequency_data"`
	RT60          *float64               `json:"rt60,omitempty"`
	RoomModes     []float64              `json:"room_modes,omitempty"`
	Metrics       map[string]interface{} `json:"metrics,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// AIInteraction represents a cached AI question/answer pair
type AIInteraction struct {
	ID           string    `json:"id"`
	AnalysisID   *string   `json:"analysis_id,omitempty"`
	QuestionHash string    `json:"question_hash"`
	Question     string    `json:"question"`
	Answer       string    `json:"answer"`
	ModelUsed    string    `json:"model_used"`
	CreatedAt    time.Time `json:"created_at"`
}
