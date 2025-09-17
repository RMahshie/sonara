# Sonara - Project Context for Cursor AI

## Project Overview

**Name:** Sonara
**Tagline:** "See your sound clearly"
**Purpose:** Web-based room acoustic analyzer using USB microphones
**Target Users:** Audio enthusiasts, home studio owners, audiophiles
**Unique Value:** Makes professional acoustic analysis accessible using consumer USB mics

## Technical Stack

### Backend
- **Language:** Go 1.21+
- **Framework:** Huma v2 (OpenAPI-first) + Chi router
- **Database:** PostgreSQL 15 with JSONB
- **File Storage:** AWS S3 with pre-signed URLs
- **Audio Processing:** Python subprocess (numpy, scipy, librosa)
- **AI:** OpenAI GPT-3.5/4
- **Testing:** testify, testcontainers-go, 80% coverage target

### Frontend
- **Framework:** React 18 with TypeScript
- **Build Tool:** Vite
- **Styling:** Tailwind CSS with custom British Racing Green theme
- **State Management:** Zustand
- **Charts:** Recharts
- **HTTP Client:** Axios
- **Testing:** Vitest, React Testing Library

### Deployment
- **Platform:** Railway
- **Domain:** sonara.app
- **SSL:** Automatic via Railway
- **Environment:** Production and development

## Project Structure

```
sonara/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/               # HTTP handlers
│   │   ├── middleware/             # HTTP middleware
│   │   └── routes.go               # Route definitions
│   ├── repository/
│   │   ├── interfaces.go           # Repository interfaces
│   │   └── postgres/               # PostgreSQL implementations
│   ├── processing/
│   │   └── service.go              # Audio processing orchestration
│   ├── storage/
│   │   └── s3.go                   # S3 service
│   └── ai/
│       └── openai.go               # OpenAI integration
├── pkg/
│   └── models/                     # Shared models
├── scripts/
│   ├── analyze_audio.py            # Python FFT analysis
│   └── requirements.txt            # Python dependencies
├── web/
│   ├── src/
│   │   ├── components/             # React components
│   │   ├── hooks/                  # Custom React hooks
│   │   ├── services/               # API clients
│   │   ├── stores/                 # Zustand stores
│   │   └── types/                  # TypeScript types
│   └── package.json
├── migrations/                      # Database migrations
├── docker/
│   └── Dockerfile
└── Makefile                        # Build commands
```

## Visual Design System

### Color Palette (British Racing Green Theme)
```css
:root {
  --racing-green: #004225;     /* Primary - British Racing Green */
  --brass: #b8860b;            /* Secondary - Brass accents */
  --cream: #fef3c7;            /* Highlights */
  --background: #fafaf9;       /* Off-white with warm tint */
  --foreground: #1c1917;       /* Deep brown-black text */
  --muted: #e7e5e4;           /* Warm gray for borders */
  --success: #059669;         /* Emerald green */
  --warning: #d97706;         /* Amber */
  --error: #dc2626;           /* Red */
}
```

### Typography
```css
/* Headers use serif for classic feel */
font-family: 'Playfair Display', serif;

/* Body text uses clean sans-serif */
font-family: 'Inter', sans-serif;
```

### Tailwind Config Extension
```javascript
module.exports = {
  theme: {
    extend: {
      colors: {
        'racing-green': '#004225',
        'brass': '#b8860b',
        'cream': '#fef3c7'
      }
    }
  }
}
```

## API Endpoints (Huma + Chi)

### Main Application Setup
```go
package main

import (
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/danielgtaylor/huma/v2"
    "github.com/danielgtaylor/huma/v2/adapters/humachi"
)

func main() {
    router := chi.NewRouter()
    
    // Middleware
    router.Use(middleware.RequestID)
    router.Use(middleware.RealIP)
    router.Use(middleware.Logger)
    router.Use(middleware.Recoverer)
    router.Use(middleware.Compress(5))
    
    // Create Huma API
    config := huma.DefaultConfig("Sonara API", "1.0.0")
    api := humachi.New(router, config)
    
    // Register routes
    RegisterAnalysisRoutes(api)
    
    // Serve OpenAPI spec
    router.Get("/api/docs", api.OpenAPI())
    
    http.ListenAndServe(":8080", router)
}
```

### POST /api/analyses - Create Analysis
```go
type CreateAnalysisInput struct {
    SessionID string `json:"session_id" minLength:"10" maxLength:"50" required:"true"`
    FileSize  int64  `json:"file_size" minimum:"1000" maximum:"20971520" required:"true"`
    MimeType  string `json:"mime_type" enum:"audio/wav,audio/mpeg,audio/flac" required:"true"`
}

type CreateAnalysisOutput struct {
    ID        string `json:"id" doc:"Analysis ID"`
    UploadURL string `json:"upload_url" doc:"Pre-signed S3 URL for upload"`
    ExpiresIn int    `json:"expires_in" doc:"URL expiration in seconds"`
}

// Handler implementation
func CreateAnalysis(ctx context.Context, input *CreateAnalysisInput) (*CreateAnalysisOutput, error) {
    // 1. Create analysis record in database
    // 2. Generate S3 pre-signed URL
    // 3. Return analysis ID and upload URL
}
```

### GET /api/analyses/{id}/status - Get Progress
```go
type GetAnalysisStatusOutput struct {
    ID       string  `json:"id"`
    Status   string  `json:"status" enum:"pending,processing,completed,failed"`
    Progress int     `json:"progress" minimum:"0" maximum:"100"`
    Message  string  `json:"message,omitempty"`
    ResultsID *string `json:"results_id,omitempty"`
}
```

### GET /api/analyses/{id}/results - Get Results
```go
type AnalysisResultsOutput struct {
    ID            string           `json:"id"`
    FrequencyData []FrequencyPoint `json:"frequency_data"`
    RT60          *float64         `json:"rt60,omitempty"`
    RoomModes     []float64        `json:"room_modes,omitempty"`
    RoomInfo      *RoomInfo        `json:"room_info,omitempty"`
    CreatedAt     time.Time        `json:"created_at"`
}

type FrequencyPoint struct {
    Frequency float64 `json:"frequency" doc:"Frequency in Hz"`
    Magnitude float64 `json:"magnitude" doc:"Magnitude in dB"`
}
```

### POST /api/analyses/{id}/room-info - Add Room Description
```go
type RoomInfoInput struct {
    RoomSize         string   `json:"room_size" enum:"small,medium,large,very_large"`
    CeilingHeight    string   `json:"ceiling_height" enum:"standard,high,vaulted"`
    FloorType        string   `json:"floor_type" enum:"carpet,hardwood,tile,rug_on_hard"`
    Features         []string `json:"features" doc:"Room features like windows, curtains, panels"`
    SpeakerPlacement string   `json:"speaker_placement" enum:"desk,stands,shelf,wall"`
    AdditionalNotes  string   `json:"additional_notes" maxLength:"500"`
}
```

### GET /api/analyses/{id}/pdf - Download PDF
```go
// Returns PDF with:
// - Frequency response chart
// - RT60 measurement
// - Room modes
// - AI recommendations
// Headers: Content-Type: application/pdf
```

### POST /api/ai/ask - Ask AI Question
```go
type AskQuestionInput struct {
    AnalysisID string `json:"analysis_id" required:"true"`
    Question   string `json:"question" minLength:"10" maxLength:"500" required:"true"`
}

type AskQuestionOutput struct {
    Answer string `json:"answer"`
    Cached bool   `json:"cached" doc:"Whether response was cached"`
}
```

## Database Schema

### PostgreSQL Tables
```sql
-- Main analysis tracking
CREATE TABLE analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    progress INTEGER DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    audio_s3_key VARCHAR(255),
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    
    INDEX idx_session_id (session_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at DESC)
);

-- Analysis results (JSONB for flexibility)
CREATE TABLE analysis_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
    frequency_data JSONB NOT NULL,  -- Array of {frequency, magnitude}
    rt60 FLOAT,
    room_modes JSONB,  -- Array of frequencies
    metrics JSONB,     -- Additional metrics
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_analysis_id (analysis_id)
);

-- Room information
CREATE TABLE room_info (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
    room_size VARCHAR(50),
    ceiling_height VARCHAR(50),
    floor_type VARCHAR(50),
    features JSONB,
    speaker_placement VARCHAR(100),
    additional_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- AI interactions cache
CREATE TABLE ai_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID REFERENCES analyses(id),
    question_hash VARCHAR(64),  -- SHA256 of question + context
    question TEXT,
    answer TEXT,
    model_used VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_question_hash (question_hash)
);
```

## Repository Pattern

### Interface Definition
```go
package repository

type AnalysisRepository interface {
    Create(ctx context.Context, analysis *models.Analysis) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.Analysis, error)
    GetBySessionID(ctx context.Context, sessionID string) ([]*models.Analysis, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status string, progress int) error
    UpdateError(ctx context.Context, id uuid.UUID, errorMsg string) error
    StoreResults(ctx context.Context, results *models.AnalysisResults) error
    GetResults(ctx context.Context, analysisID uuid.UUID) (*models.AnalysisResults, error)
}

type RoomInfoRepository interface {
    Create(ctx context.Context, info *models.RoomInfo) error
    GetByAnalysisID(ctx context.Context, analysisID uuid.UUID) (*models.RoomInfo, error)
}
```

### PostgreSQL Implementation
```go
package postgres

type postgresAnalysisRepo struct {
    db *sql.DB
}

func (r *postgresAnalysisRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, progress int) error {
    query := `
        UPDATE analyses 
        SET status = $1, progress = $2, updated_at = NOW()
        WHERE id = $3
    `
    _, err := r.db.ExecContext(ctx, query, status, progress, id)
    return err
}
```

## S3 Service

### Interface and Implementation
```go
package storage

type S3Service interface {
    GenerateUploadURL(ctx context.Context, key string, contentType string) (string, error)
    GenerateDownloadURL(ctx context.Context, key string) (string, error)
    DownloadFile(ctx context.Context, key string) ([]byte, error)
    DeleteFile(ctx context.Context, key string) error
}

type s3Service struct {
    client    *s3.Client
    bucket    string
    urlExpiry time.Duration  // 15 minutes
}

func (s *s3Service) GenerateUploadURL(ctx context.Context, key string, contentType string) (string, error) {
    // Validate content type
    validTypes := map[string]bool{
        "audio/wav":  true,
        "audio/mpeg": true,
        "audio/flac": true,
    }
    
    if !validTypes[contentType] {
        return "", fmt.Errorf("invalid content type: %s", contentType)
    }
    
    presignClient := s3.NewPresignClient(s.client)
    request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
        Bucket:      &s.bucket,
        Key:         &key,
        ContentType: &contentType,
    }, func(opts *s3.PresignOptions) {
        opts.Expires = s.urlExpiry
    })
    
    return request.URL, nil
}
```

## Audio Processing (Python)

### Main Analysis Script
```python
# scripts/analyze_audio.py
import sys
import json
import numpy as np
from scipy import signal
from scipy.io import wavfile
import librosa

class AudioAnalyzer:
    def __init__(self):
        # FIFINE K669 USB Microphone calibration curve
        # Compensates for microphone frequency response
        self.calibration_curve = [
            (20, 12),     # +12dB at 20Hz (mic rolls off)
            (50, 3),      # +3dB at 50Hz
            (100, 0),     # Flat at 100Hz
            (200, 0),     # Flat at 200Hz
            (500, 0),     # Flat at 500Hz
            (1000, 0),    # Flat at 1kHz (reference)
            (2000, -1),   # -1dB at 2kHz
            (5000, -2),   # -2dB at 5kHz
            (8000, -3),   # -3dB at 8kHz (mic boost)
            (10000, -3.5), # -3.5dB at 10kHz
            (12000, -4),  # -4dB at 12kHz
            (16000, -2),  # -2dB at 16kHz
            (20000, 5)    # +5dB at 20kHz (mic rolls off)
        ]
    
    def load_audio(self, filepath):
        """Load audio file and convert to mono"""
        # librosa handles WAV, MP3, FLAC
        audio, sr = librosa.load(filepath, sr=None, mono=True)
        return audio, sr
    
    def perform_fft(self, audio, sample_rate):
        """
        Perform FFT analysis with proper windowing
        Returns frequency and magnitude arrays
        """
        # Apply Hamming window to reduce spectral leakage
        windowed = audio * signal.windows.hamming(len(audio))
        
        # Perform FFT (real FFT since audio is real)
        fft_result = np.fft.rfft(windowed)
        magnitude = np.abs(fft_result)
        
        # Convert to decibels (20*log10)
        magnitude_db = 20 * np.log10(magnitude + 1e-10)  # Small value to avoid log(0)
        
        # Generate frequency bins
        frequencies = np.fft.rfftfreq(len(audio), 1/sample_rate)
        
        return frequencies, magnitude_db
    
    def apply_calibration(self, frequencies, magnitudes):
        """Apply FIFINE K669 calibration curve"""
        # Extract calibration points
        cal_freqs, cal_values = zip(*self.calibration_curve)
        
        # Interpolate calibration values for all frequencies
        calibration = np.interp(frequencies, cal_freqs, cal_values)
        
        # Apply calibration (add correction values)
        calibrated_magnitudes = magnitudes + calibration
        
        return calibrated_magnitudes
    
    def calculate_rt60(self, audio, sample_rate):
        """
        Calculate RT60 (reverberation time) using Schroeder integration
        Returns time in seconds for 60dB decay
        """
        # This is a simplified implementation
        # Real RT60 requires impulse response or interrupted noise
        # For now, return placeholder
        return 0.5  # Will be properly implemented in Week 2
    
    def detect_room_modes(self, frequencies, magnitudes):
        """
        Detect room modes (resonant frequencies)
        Returns list of problematic frequencies
        """
        # Find peaks in frequency response
        # Simplified for Week 1, enhanced in Week 2
        from scipy.signal import find_peaks
        
        # Find peaks that are >6dB above neighbors
        peaks, properties = find_peaks(magnitudes, prominence=6, distance=20)
        
        # Return frequencies of peaks below 300Hz (typical room mode range)
        room_modes = []
        for peak in peaks:
            if frequencies[peak] < 300:
                room_modes.append(float(frequencies[peak]))
        
        return room_modes[:5]  # Return top 5 modes
    
    def analyze(self, filepath):
        """Main analysis function"""
        try:
            # Load audio
            audio, sample_rate = self.load_audio(filepath)
            
            # Perform FFT
            frequencies, magnitudes = self.perform_fft(audio, sample_rate)
            
            # Apply calibration
            calibrated = self.apply_calibration(frequencies, magnitudes)
            
            # Calculate RT60 (placeholder for Week 1)
            rt60 = self.calculate_rt60(audio, sample_rate)
            
            # Detect room modes
            room_modes = self.detect_room_modes(frequencies, calibrated)
            
            # Reduce data points for reasonable JSON size
            # Take every Nth point to get ~1000 points
            step = max(1, len(frequencies) // 1000)
            
            # Filter to audible range (20Hz - 20kHz)
            frequency_data = []
            for i in range(0, len(frequencies), step):
                if 20 <= frequencies[i] <= 20000:
                    frequency_data.append({
                        "frequency": float(frequencies[i]),
                        "magnitude": float(calibrated[i])
                    })
            
            result = {
                "sample_rate": int(sample_rate),
                "frequency_data": frequency_data,
                "rt60": rt60,
                "room_modes": room_modes
            }
            
            return result
            
        except Exception as e:
            return {"error": str(e)}

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(json.dumps({"error": "Usage: python analyze_audio.py <audio_file>"}))
        sys.exit(1)
    
    analyzer = AudioAnalyzer()
    result = analyzer.analyze(sys.argv[1])
    print(json.dumps(result))
    
    if "error" in result:
        sys.exit(1)
```

## Processing Pipeline

### Processing Service
```go
package processing

type ProcessingService interface {
    ProcessAnalysis(ctx context.Context, analysisID uuid.UUID) error
}

type processingService struct {
    s3         storage.S3Service
    repository repository.AnalysisRepository
    pythonPath string  // "scripts/analyze_audio.py"
}

func (s *processingService) ProcessAnalysis(ctx context.Context, analysisID uuid.UUID) error {
    // Update progress throughout processing
    progressUpdates := []struct {
        progress int
        message  string
    }{
        {10, "Starting analysis..."},
        {20, "Downloading audio file..."},
        {50, "Analyzing frequency response..."},
        {80, "Calculating room characteristics..."},
        {90, "Finalizing results..."},
        {100, "Analysis complete!"},
    }
    
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
    s.repository.UpdateStatus(ctx, analysisID, "processing", 20)
    audioData, err := s.s3.DownloadFile(ctx, analysis.AudioS3Key)
    if err != nil {
        s.repository.UpdateError(ctx, analysisID, "Failed to download audio")
        return err
    }
    
    // Step 4: Save to temp file
    tempFile := filepath.Join("/tmp", fmt.Sprintf("%s.audio", analysisID))
    if err := os.WriteFile(tempFile, audioData, 0644); err != nil {
        return err
    }
    defer os.Remove(tempFile)  // Always cleanup
    
    // Step 5: Run Python analysis
    s.repository.UpdateStatus(ctx, analysisID, "processing", 50)
    cmd := exec.CommandContext(ctx, "python3", s.pythonPath, tempFile)
    output, err := cmd.Output()
    if err != nil {
        s.repository.UpdateError(ctx, analysisID, "Audio analysis failed")
        return fmt.Errorf("python analysis failed: %w", err)
    }
    
    // Step 6: Parse results
    s.repository.UpdateStatus(ctx, analysisID, "processing", 80)
    var result struct {
        FrequencyData []FrequencyPoint `json:"frequency_data"`
        RT60          float64          `json:"rt60"`
        RoomModes     []float64        `json:"room_modes"`
        Error         string           `json:"error,omitempty"`
    }
    
    if err := json.Unmarshal(output, &result); err != nil {
        return fmt.Errorf("failed to parse results: %w", err)
    }
    
    if result.Error != "" {
        s.repository.UpdateError(ctx, analysisID, result.Error)
        return fmt.Errorf("analysis error: %s", result.Error)
    }
    
    // Step 7: Store results
    s.repository.UpdateStatus(ctx, analysisID, "processing", 90)
    if err := s.repository.StoreResults(ctx, &models.AnalysisResults{
        AnalysisID:    analysisID,
        FrequencyData: result.FrequencyData,
        RT60:          result.RT60,
        RoomModes:     result.RoomModes,
    }); err != nil {
        return err
    }
    
    // Step 8: Mark complete
    if err := s.repository.UpdateStatus(ctx, analysisID, "completed", 100); err != nil {
        return err
    }
    
    return nil
}
```

## React Components

### File Upload Component
```tsx
// web/src/components/FileUpload.tsx
import React, { useState, useCallback } from 'react';
import { useDropzone } from 'react-dropzone';
import axios from 'axios';
import { useNavigate } from 'react-router-dom';

interface FileUploadProps {
    onUploadStart?: (analysisId: string) => void;
}

export const FileUpload: React.FC<FileUploadProps> = ({ onUploadStart }) => {
    const [uploading, setUploading] = useState(false);
    const [progress, setProgress] = useState(0);
    const [error, setError] = useState<string | null>(null);
    const navigate = useNavigate();
    
    const validateFile = (file: File): string | null => {
        const validTypes = ['audio/wav', 'audio/mpeg', 'audio/mp3', 'audio/flac', 'audio/x-flac'];
        const validExtensions = ['.wav', '.mp3', '.flac'];
        
        // Check MIME type
        if (!validTypes.includes(file.type)) {
            // Also check extension as fallback
            const ext = file.name.toLowerCase().slice(file.name.lastIndexOf('.'));
            if (!validExtensions.includes(ext)) {
                return 'Please upload a WAV, MP3, or FLAC file';
            }
        }
        
        // Check file size (20MB max)
        if (file.size > 20 * 1024 * 1024) {
            return 'File size must be less than 20MB';
        }
        
        return null;
    };
    
    const uploadFile = async (file: File) => {
        setUploading(true);
        setError(null);
        setProgress(0);
        
        try {
            // Get or create session ID
            let sessionId = localStorage.getItem('sonara_session_id');
            if (!sessionId) {
                sessionId = `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
                localStorage.setItem('sonara_session_id', sessionId);
            }
            
            // Step 1: Create analysis and get upload URL
            const createResponse = await axios.post('/api/analyses', {
                session_id: sessionId,
                file_size: file.size,
                mime_type: file.type || 'audio/wav'
            });
            
            const { id: analysisId, upload_url } = createResponse.data;
            
            // Step 2: Upload directly to S3
            await axios.put(upload_url, file, {
                headers: {
                    'Content-Type': file.type || 'audio/wav'
                },
                onUploadProgress: (progressEvent) => {
                    const percent = Math.round((progressEvent.loaded * 100) / progressEvent.total!);
                    setProgress(percent);
                }
            });
            
            // Step 3: Navigate to analysis page
            if (onUploadStart) {
                onUploadStart(analysisId);
            }
            navigate(`/analysis/${analysisId}`);
            
        } catch (err) {
            console.error('Upload error:', err);
            setError('Upload failed. Please try again.');
        } finally {
            setUploading(false);
        }
    };
    
    const onDrop = useCallback((acceptedFiles: File[]) => {
        if (acceptedFiles.length === 0) return;
        
        const file = acceptedFiles[0];
        const validationError = validateFile(file);
        
        if (validationError) {
            setError(validationError);
            return;
        }
        
        uploadFile(file);
    }, []);
    
    const { getRootProps, getInputProps, isDragActive } = useDropzone({
        onDrop,
        accept: {
            'audio/*': ['.wav', '.mp3', '.flac']
        },
        maxFiles: 1,
        disabled: uploading
    });
    
    return (
        <div className="w-full max-w-xl mx-auto">
            <div
                {...getRootProps()}
                className={`
                    border-2 border-dashed rounded-lg p-8 text-center cursor-pointer
                    transition-all duration-200
                    ${isDragActive ? 'border-racing-green bg-cream/20' : 'border-gray-300'}
                    ${uploading ? 'opacity-50 cursor-not-allowed' : 'hover:border-racing-green'}
                `}
            >
                <input {...getInputProps()} />
                
                {!uploading && (
                    <>
                        <svg 
                            className="mx-auto h-12 w-12 text-gray-400 mb-4" 
                            fill="none" 
                            viewBox="0 0 24 24" 
                            stroke="currentColor"
                        >
                            <path 
                                strokeLinecap="round" 
                                strokeLinejoin="round" 
                                strokeWidth={2} 
                                d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" 
                            />
                        </svg>
                        <p className="text-lg font-medium text-gray-700">
                            {isDragActive ? 'Drop your audio file here' : 'Drag & drop your audio file'}
                        </p>
                        <p className="text-sm text-gray-500 mt-2">
                            or click to browse
                        </p>
                        <p className="text-xs text-gray-400 mt-2">
                            WAV, MP3, or FLAC • Max 20MB
                        </p>
                    </>
                )}
                
                {uploading && (
                    <div className="space-y-4">
                        <p className="text-sm font-medium">Uploading...</p>
                        <div className="w-full bg-gray-200 rounded-full h-2">
                            <div
                                className="bg-racing-green h-2 rounded-full transition-all duration-300"
                                style={{ width: `${progress}%` }}
                            />
                        </div>
                        <p className="text-sm text-gray-600">{progress}%</p>
                    </div>
                )}
            </div>
            
            {error && (
                <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-md">
                    <p className="text-sm text-red-600">{error}</p>
                </div>
            )}
        </div>
    );
};
```

### Analysis Status Hook
```tsx
// web/src/hooks/useAnalysisStatus.ts
import { useState, useEffect, useCallback } from 'react';
import axios from 'axios';

interface AnalysisStatus {
    id: string;
    status: 'pending' | 'processing' | 'completed' | 'failed';
    progress: number;
    message?: string;
    resultsId?: string;
}

export function useAnalysisStatus(analysisId: string | null) {
    const [status, setStatus] = useState<AnalysisStatus | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [isPolling, setIsPolling] = useState(true);
    
    const fetchStatus = useCallback(async () => {
        if (!analysisId) return;
        
        try {
            const response = await axios.get(`/api/analyses/${analysisId}/status`);
            setStatus(response.data);
            
            // Stop polling if completed or failed
            if (response.data.status === 'completed' || response.data.status === 'failed') {
                setIsPolling(false);
            }
        } catch (err) {
            console.error('Failed to fetch status:', err);
            setError('Failed to fetch analysis status');
            setIsPolling(false);
        }
    }, [analysisId]);
    
    useEffect(() => {
        if (!analysisId || !isPolling) return;
        
        // Initial fetch
        fetchStatus();
        
        // Poll every 2 seconds
        const interval = setInterval(fetchStatus, 2000);
        
        return () => clearInterval(interval);
    }, [analysisId, isPolling, fetchStatus]);
    
    return {
        status,
        error,
        isComplete: status?.status === 'completed',
        isFailed: status?.status === 'failed',
        progress: status?.progress || 0,
        message: status?.message
    };
}
```

### Frequency Response Chart
```tsx
// web/src/components/FrequencyChart.tsx
import React from 'react';
import {
    LineChart,
    Line,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    Legend,
    ResponsiveContainer
} from 'recharts';

interface FrequencyPoint {
    frequency: number;
    magnitude: number;
}

interface FrequencyChartProps {
    data: FrequencyPoint[];
    width?: number | string;
    height?: number;
}

export const FrequencyChart: React.FC<FrequencyChartProps> = ({ 
    data, 
    width = '100%', 
    height = 400 
}) => {
    // Format data for Recharts
    const chartData = data.map(point => ({
        freq: point.frequency,
        db: point.magnitude
    }));
    
    // Custom tick formatter for frequency axis
    const formatFrequency = (value: number) => {
        if (value >= 1000) {
            return `${(value / 1000).toFixed(0)}k`;
        }
        return value.toString();
    };
    
    return (
        <ResponsiveContainer width={width} height={height}>
            <LineChart 
                data={chartData}
                margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
            >
                <CartesianGrid strokeDasharray="3 3" stroke="#e7e5e4" />
                <XAxis 
                    dataKey="freq"
                    scale="log"
                    domain={[20, 20000]}
                    ticks={[20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000]}
                    tickFormatter={formatFrequency}
                    label={{ value: 'Frequency (Hz)', position: 'insideBottom', offset: -5 }}
                    stroke="#1c1917"
                />
                <YAxis 
                    domain={[-30, 10]}
                    label={{ value: 'Magnitude (dB)', angle: -90, position: 'insideLeft' }}
                    stroke="#1c1917"
                />
                <Tooltip 
                    formatter={(value: number) => `${value.toFixed(1)} dB`}
                    labelFormatter={(value: number) => `${value.toFixed(0)} Hz`}
                    contentStyle={{ 
                        backgroundColor: '#fafaf9',
                        border: '2px solid #004225',
                        borderRadius: '4px'
                    }}
                />
                <Legend />
                <Line 
                    type="monotone" 
                    dataKey="db" 
                    stroke="#004225"
                    strokeWidth={2}
                    dot={false}
                    name="Frequency Response"
                />
            </LineChart>
        </ResponsiveContainer>
    );
};
```

## OpenAI Integration

### AI Service
```go
package ai

type OpenAIService interface {
    GetRecommendations(ctx context.Context, analysisContext AnalysisContext) (string, error)
    AskQuestion(ctx context.Context, question string, analysisContext AnalysisContext) (string, error)
}

type openAIService struct {
    client   *openai.Client
    cache    map[string]cachedResponse  // Simple in-memory cache
    cacheMux sync.RWMutex
}

type AnalysisContext struct {
    FrequencyData []FrequencyPoint
    RT60          float64
    RoomModes     []float64
    RoomInfo      *RoomInfo
}

func (s *openAIService) buildPrompt(ctx AnalysisContext) string {
    return fmt.Sprintf(`You are an expert acoustic engineer analyzing room measurements.

Room Measurements:
- RT60 (Reverb Time): %.2f seconds
- Room Modes: %v Hz
- Room Size: %s
- Floor Type: %s
- Ceiling: %s

The frequency response shows:
- Bass response (20-200Hz): [Analyze from data]
- Midrange (200-2000Hz): [Analyze from data]
- Treble (2000-20000Hz): [Analyze from data]

Provide specific, actionable recommendations for acoustic treatment.
Focus on practical solutions that don't require construction.
Prioritize by impact and cost-effectiveness.`,
        ctx.RT60,
        ctx.RoomModes,
        ctx.RoomInfo.RoomSize,
        ctx.RoomInfo.FloorType,
        ctx.RoomInfo.CeilingHeight,
    )
}

func (s *openAIService) GetRecommendations(ctx context.Context, analysisCtx AnalysisContext) (string, error) {
    // Check cache first
    cacheKey := s.generateCacheKey(analysisCtx)
    if cached, ok := s.getCached(cacheKey); ok {
        return cached.response, nil
    }
    
    // Build messages
    messages := []openai.ChatCompletionMessage{
        {
            Role:    openai.ChatMessageRoleSystem,
            Content: "You are an expert acoustic engineer specializing in room treatment.",
        },
        {
            Role:    openai.ChatMessageRoleUser,
            Content: s.buildPrompt(analysisCtx),
        },
    }
    
    // Call OpenAI
    resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:       openai.GPT3Dot5Turbo,  // Use GPT-4 for complex questions
        Messages:    messages,
        Temperature: 0.7,
        MaxTokens:   500,
    })
    
    if err != nil {
        return "", fmt.Errorf("OpenAI API error: %w", err)
    }
    
    answer := resp.Choices[0].Message.Content
    
    // Cache response
    s.setCached(cacheKey, answer)
    
    return answer, nil
}
```

## Testing Patterns

### Backend Testing with Testify
```go
// internal/api/handlers/analysis_test.go
package handlers

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type MockAnalysisRepository struct {
    mock.Mock
}

func (m *MockAnalysisRepository) Create(ctx context.Context, analysis *models.Analysis) error {
    args := m.Called(ctx, analysis)
    return args.Error(0)
}

func TestCreateAnalysis(t *testing.T) {
    tests := []struct {
        name      string
        input     CreateAnalysisInput
        wantCode  int
        wantError bool
    }{
        {
            name: "valid audio file",
            input: CreateAnalysisInput{
                SessionID: "test-session-123",
                FileSize:  5242880,  // 5MB
                MimeType:  "audio/wav",
            },
            wantCode:  201,
            wantError: false,
        },
        {
            name: "file too large",
            input: CreateAnalysisInput{
                SessionID: "test-session-123",
                FileSize:  25000000,  // 25MB
                MimeType:  "audio/wav",
            },
            wantCode:  400,
            wantError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Frontend Testing with React Testing Library
```tsx
// web/src/components/__tests__/FileUpload.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FileUpload } from '../FileUpload';

describe('FileUpload', () => {
    it('accepts valid audio files', async () => {
        const file = new File(['audio content'], 'test.wav', { 
            type: 'audio/wav' 
        });
        
        render(<FileUpload />);
        
        const input = screen.getByLabelText(/drag.*drop/i);
        await userEvent.upload(input, file);
        
        expect(screen.queryByText(/error/i)).not.toBeInTheDocument();
    });
    
    it('shows upload progress', async () => {
        // Mock axios
        // Test progress updates
    });
});
```

## Environment Variables

### Development (.env.development)
```bash
# API
API_URL=http://localhost:8080

# Database
DATABASE_URL=postgres://user:password@localhost:5432/sonara_dev

# AWS S3
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_dev_key
AWS_SECRET_ACCESS_KEY=your_dev_secret
S3_BUCKET=sonara-dev-audio

# OpenAI
OPENAI_API_KEY=sk-...

# Server
PORT=8080
ENVIRONMENT=development
```

### Production (.env.production)
```bash
# Provided by Railway
DATABASE_URL=${DATABASE_URL}
PORT=${PORT}

# Set in Railway dashboard
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}
AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}
S3_BUCKET=sonara-audio-files
OPENAI_API_KEY=${OPENAI_API_KEY}
ENVIRONMENT=production
```

## Makefile Commands

```makefile
# Development
.PHONY: dev
dev:
	air -c .air.toml  # Hot reload for Go

.PHONY: dev-web
dev-web:
	cd web && pnpm dev

# Database
.PHONY: migrate-up
migrate-up:
	migrate -path migrations -database $(DATABASE_URL) up

.PHONY: migrate-down
migrate-down:
	migrate -path migrations -database $(DATABASE_URL) down

.PHONY: migrate-create
migrate-create:
	migrate create -ext sql -dir migrations $(name)

# Testing
.PHONY: test
test:
	go test ./... -v -cover

.PHONY: test-web
test-web:
	cd web && pnpm test

# Building
.PHONY: build
build:
	go build -o bin/sonara cmd/server/main.go

.PHONY: build-web
build-web:
	cd web && pnpm build

# Docker
.PHONY: docker-build
docker-build:
	docker build -t sonara:latest .

.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 --env-file .env sonara:latest

# Production
.PHONY: deploy
deploy:
	railway up
```

## Key Implementation Notes

1. **Always use pre-signed URLs for file uploads** - Never stream through the server
2. **FIFINE K669 calibration is hardcoded** - Can add more mic profiles later
3. **Polling every 2 seconds is sufficient** - WebSockets are overkill for this
4. **Use JSONB for frequency data** - Flexible and performant
5. **Repository pattern for all database access** - Clean separation of concerns
6. **Mock all external dependencies in tests** - Fast, reliable tests
7. **British Racing Green theme throughout** - Consistent visual identity
8. **Cache AI responses** - OpenAI API is expensive
9. **Progress updates are UX critical** - Users need feedback during processing
10. **Test coverage target is 80%** - Shows production quality