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
    MimeType  string `json:"mime_type" enum:"audio/wav,audio/mpeg,audio/flac,audio/webm,audio/ogg" required:"true"`
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
    RoomLength       *float64 `json:"room_length,omitempty" doc:"Room length in meters for resonance analysis"`
    RoomWidth        *float64 `json:"room_width,omitempty" doc:"Room width in meters for resonance analysis"`
    RoomHeight       *float64 `json:"room_height,omitempty" doc:"Room height in meters for resonance analysis"`
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

    // Step 4.5: Convert WebM/OGG to WAV for Python compatibility
    wavFile := filepath.Join("/tmp", fmt.Sprintf("%s.wav", analysisID))
    convertCmd := exec.Command("ffmpeg", "-i", tempFile, "-acodec", "pcm_s16le", "-ar", "48000", "-y", wavFile)
    if err := convertCmd.Run(); err != nil {
        return fmt.Errorf("failed to convert audio to WAV: %w", err)
    }
    defer os.Remove(wavFile) // Cleanup WAV file
    
    // Step 5: Run Python analysis
    s.repository.UpdateStatus(ctx, analysisID, "processing", 50)
    cmd := exec.CommandContext(ctx, "python3", s.pythonPath, wavFile)
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

### Live Recorder Component
```tsx
// web/src/components/LiveRecorder.tsx
import React, { useState, useRef, useCallback, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { analysisService } from '../services/analysisService';
import ProgressBar from './ProgressBar';

export const LiveRecorder: React.FC = () => {
    const [isRecording, setIsRecording] = useState(false);
    const [progress, setProgress] = useState(0);
    const [error, setError] = useState<string | null>(null);
    const [phase, setPhase] = useState<'ready' | 'room-input' | 'recording' | 'processing'>('ready');
    const [roomDimensions, setRoomDimensions] = useState({
        length: '',
        width: '',
        height: ''
    });

    const mediaRecorderRef = useRef<MediaRecorder | null>(null);
    const streamRef = useRef<MediaStream | null>(null);
    const navigate = useNavigate();
    
    // Form validation for room dimensions
    const isRoomFormValid = () => {
        // All empty = valid (skip room analysis)
        if (!roomDimensions.length && !roomDimensions.width && !roomDimensions.height) {
            return true;
        }
        // All filled with valid numbers = valid
        return [roomDimensions.length, roomDimensions.width, roomDimensions.height]
            .every(dim => dim === '' || (parseFloat(dim) > 0 && parseFloat(dim) < 50));
    };

    // Entry point for room dimensions collection
    const handleAnalyzeRoom = useCallback(() => {
        setError(null);
        setPhase('room-input');
    }, []);

    // Start recording with optional room data
    const startRecordingWithRoomInfo = useCallback(() => {
        setError(null);
        setPhase('recording');
        startRecording();
    }, []);

    // Core recording logic (shared between entry points)
    const startRecording = useCallback(async () => {
        try {
            // Request microphone access with optimized settings
            const stream = await navigator.mediaDevices.getUserMedia({
                audio: {
                    echoCancellation: false,
                    noiseSuppression: false,
                    autoGainControl: false,
                    sampleRate: 48000
                }
            });
            streamRef.current = stream;

            // Setup MediaRecorder for WebM/OGG output
            const mimeType = MediaRecorder.isTypeSupported('audio/webm')
                ? 'audio/webm'
                : 'audio/ogg';

            const mediaRecorder = new MediaRecorder(stream, { mimeType });
            mediaRecorderRef.current = mediaRecorder;

            const chunks: Blob[] = [];

            mediaRecorder.ondataavailable = (event) => {
                if (event.data.size > 0) {
                    chunks.push(event.data);
                }
            };

            mediaRecorder.onstop = async () => {
                const audioBlob = new Blob(chunks, { type: mimeType });

                // Validate recording quality
                if (audioBlob.size < 1000) {
                    setError('Recording failed. Please check your microphone.');
                    setPhase('ready');
                    stream.getTracks().forEach(track => track.stop());
                    return;
                }

                await uploadRecording(audioBlob, mimeType);
                stream.getTracks().forEach(track => track.stop());
            };

            // Play pink noise test signal simultaneously with recording
            const testSignal = new Audio('/test-signals/sweep-20-20k-10s.wav');
            testSignal.volume = 1.0;

            testSignal.onerror = () => {
                setError('Test signal failed to load. Please check your setup.');
                setPhase('ready');
                setIsRecording(false);
                mediaRecorder.stop();
                stream.getTracks().forEach(track => track.stop());
                return;
            };

            testSignal.ontimeupdate = () => {
                const currentProgress = (testSignal.currentTime / testSignal.duration) * 100;
                setProgress(Math.round(currentProgress));
            };

            testSignal.onended = () => {
                mediaRecorder.stop();
                setIsRecording(false);
                setPhase('processing'); // Direct to processing (single screen)
            };

            // Start recording then play test signal
            mediaRecorder.start();
            setIsRecording(true);

            try {
                await testSignal.play();
            } catch (playError) {
                setError('Cannot play test signal. Check audio permissions.');
                setPhase('ready');
                setIsRecording(false);
                mediaRecorder.stop();
                stream.getTracks().forEach(track => track.stop());
                return;
            }

        } catch (err: any) {
            setError(err.message || 'Failed to access microphone. Please check permissions.');
            setPhase('ready');
            setIsRecording(false);
        }
    }, [navigate]);

    const uploadRecording = async (audioBlob: Blob, mimeType: string) => {
        try {
            // Get or create session ID
            let sessionId = localStorage.getItem('sonara_session_id');
            if (!sessionId) {
                sessionId = `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
                localStorage.setItem('sonara_session_id', sessionId);
            }
            
            // Convert blob to file for the service
            const file = new File([audioBlob], 'recording', { type: mimeType });

            // Create analysis and get upload URL
            const { id: analysisId, upload_url: uploadUrl } = await analysisService.createAnalysis(sessionId, file);

            // Send room dimensions if provided
            if (roomDimensions.length || roomDimensions.width || roomDimensions.height) {
                try {
                    await analysisService.addRoomInfo(analysisId, {
                        room_length: roomDimensions.length ? parseFloat(roomDimensions.length) : undefined,
                        room_width: roomDimensions.width ? parseFloat(roomDimensions.width) : undefined,
                        room_height: roomDimensions.height ? parseFloat(roomDimensions.height) : undefined,
                        room_size: 'medium', // Default values for required fields
                        ceiling_height: 'standard',
                        floor_type: 'hardwood',
                        features: [],
                        speaker_placement: 'desk',
                        additional_notes: ''
                    });
                } catch (err: any) {
                    console.warn('Failed to save room info, continuing with analysis:', err);
                    // Don't block the analysis if room info fails
                }
            }

            // Upload to S3 with progress tracking
            await analysisService.uploadToS3(uploadUrl, file, (uploadProgress) => {
                // Upload progress contributes to overall progress
                const totalProgress = Math.round(uploadProgress * 0.5);
                setProgress(totalProgress);
            });

            // Start backend processing
            setProgress(50); // Jump to processing phase

            await analysisService.startProcessing(analysisId);

            // Navigate to analysis page (processing continues in background)
            navigate(`/analysis/${analysisId}`);
            
        } catch (err: any) {
            // Use backend-provided error messages when available
            if (err.response?.data?.message) {
                setError(err.response.data.message);
            } else if (err.response?.status === 413) {
                setError('Recording file is too large for upload.');
            } else if (!navigator.onLine) {
                setError('No internet connection. Please check your network.');
            } else {
            setError('Upload failed. Please try again.');
            }
            setPhase('ready');
        }
    };

    const stopAnalysis = useCallback(() => {
        if (mediaRecorderRef.current && isRecording) {
            mediaRecorderRef.current.stop();
            setIsRecording(false);
        }
        if (streamRef.current) {
            streamRef.current.getTracks().forEach(track => track.stop());
        }
    }, [isRecording]);
    
    return (
        <div className="w-full">
            <div
                className={`
                    border-2 border-dashed rounded-xl p-12 text-center transition-all duration-200
                    ${error ? 'border-red-300 bg-red-50' : 'border-racing-green/30 hover:border-racing-green/60'}
                    ${isRecording ? 'pointer-events-none opacity-50' : ''}
                `}
            >
                {phase === 'ready' && !isRecording && (
                    <div className="space-y-4">
                        <div className="text-6xl text-racing-green/40">🎤</div>
                        <div>
                            <p className="text-xl font-medium text-racing-green mb-2">
                                Room Acoustic Analysis
                            </p>
                            <p className="text-racing-green/60 mb-4">
                                Click to analyze your room acoustics
                            </p>
                            <button
                                onClick={handleAnalyzeRoom}
                                className="btn-primary mt-4"
                                disabled={isRecording || phase !== 'ready'}
                            >
                                Analyze Room
                            </button>
                        </div>
                    </div>
                )}

                {phase === 'room-input' && (
                    <div className="space-y-4">
                        <div className="text-6xl text-racing-green/40">📏</div>
                        <div>
                            <p className="text-xl font-medium text-racing-green mb-2">
                                Room Dimensions (Optional)
                            </p>
                            <p className="text-racing-green/60 mb-4">
                                Enter your room dimensions for enhanced resonance analysis
                            </p>

                            <div className="space-y-3 mb-6">
                                <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                                    <div>
                                        <label className="block text-sm text-racing-green/70 mb-1">
                                            Length (m)
                                        </label>
                                        <input
                                            type="number"
                                            step="0.1"
                                            min="0.1"
                                            max="50"
                                            placeholder="e.g. 4.5"
                                            className="w-full px-3 py-2 border border-racing-green/30 rounded-md focus:outline-none focus:ring-2 focus:ring-racing-green/50"
                                            value={roomDimensions.length}
                                            onChange={(e) => setRoomDimensions(prev => ({...prev, length: e.target.value}))}
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm text-racing-green/70 mb-1">
                                            Width (m)
                                        </label>
                                        <input
                                            type="number"
                                            step="0.1"
                                            min="0.1"
                                            max="50"
                                            placeholder="e.g. 3.2"
                                            className="w-full px-3 py-2 border border-racing-green/30 rounded-md focus:outline-none focus:ring-2 focus:ring-racing-green/50"
                                            value={roomDimensions.width}
                                            onChange={(e) => setRoomDimensions(prev => ({...prev, width: e.target.value}))}
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm text-racing-green/70 mb-1">
                                            Height (m)
                                        </label>
                                        <input
                                            type="number"
                                            step="0.1"
                                            min="0.1"
                                            max="10"
                                            placeholder="e.g. 2.4"
                                            className="w-full px-3 py-2 border border-racing-green/30 rounded-md focus:outline-none focus:ring-2 focus:ring-racing-green/50"
                                            value={roomDimensions.height}
                                            onChange={(e) => setRoomDimensions(prev => ({...prev, height: e.target.value}))}
                                        />
                                    </div>
                                </div>
                                <p className="text-xs text-racing-green/50">
                                    Leave blank to skip enhanced analysis
                                </p>
                            </div>

                            <div className="flex gap-3">
                                <button
                                    onClick={() => setPhase('ready')}
                                    className="px-4 py-2 text-racing-green/70 hover:text-racing-green border border-racing-green/30 rounded-md transition-colors"
                                >
                                    Back
                                </button>
                                <button
                                    onClick={startRecordingWithRoomInfo}
                                    className="btn-primary flex-1"
                                    disabled={!isRoomFormValid()}
                                >
                                    Start Analysis
                                </button>
                            </div>
                        </div>
                    </div>
                )}

                {phase === 'recording' && isRecording && (
                    <div className="space-y-4">
                        <div className="text-6xl text-racing-green/40 animate-pulse">🔴</div>
                        <div className="text-racing-green font-medium">
                            Recording in progress...
                        </div>
                        <ProgressBar progress={progress} />
                        <p className="text-sm text-racing-green/60">
                            Please remain quiet during the measurement
                        </p>
                        <button
                            onClick={stopAnalysis}
                            className="text-red-600 hover:text-red-700 text-sm"
                        >
                            Cancel Recording
                        </button>
                    </div>
                )}

                {phase === 'processing' && (
                    <div className="space-y-4">
                        <div className="text-6xl text-racing-green/40">⏳</div>
                        <div className="text-racing-green font-medium">
                            Processing your recording...
            </div>
                        <ProgressBar progress={progress} />
                        <p className="text-sm text-racing-green/60">
                            Analyzing frequency response and room characteristics...
                        </p>
                    </div>
                )}
            
            {error && (
                    <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-lg">
                        <p className="text-red-600 text-sm">{error}</p>
                        <button
                            onClick={() => {
                                setError(null);
                                setPhase('ready');
                            }}
                            className="mt-2 text-red-600 hover:text-red-700 text-sm underline"
                        >
                            Try Again
                        </button>
                </div>
            )}

                {phase === 'ready' && !error && (
                    <div className="mt-8 text-xs text-racing-green/60 max-w-md mx-auto">
                        <p className="mb-2 font-medium text-base">Quick Setup:</p>
                        <ul className="space-y-1">
                            <li>• Set speakers to normal listening volume</li>
                            <li>• Position microphone at listening position</li>
                            <li>• Minimize background noise</li>
                        </ul>
                    </div>
                )}
            </div>
        </div>
    );
};
```

### Analysis Status Hook
```tsx
// web/src/hooks/useAnalysisStatus.ts
import { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

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
            const response = await api.get(`/analyses/${analysisId}/status`);
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
interface FrequencyData {
    frequency: number;
    magnitude: number;
}

interface FrequencyChartProps {
    data: FrequencyData[];
}

const FrequencyChart = ({ data }: FrequencyChartProps) => {
    // Chart dimensions
    const width = 800;
    const height = 400;
    const padding = 60;

    // Fixed ranges for professional audio visualization
    const FREQ_MIN = 20;
    const FREQ_MAX = 20000;
    const DB_MIN = -15;
    const DB_MAX = 15;

    // Scaling functions
    const xScale = (freq: number) => {
        const logFreq = Math.log10(Math.max(FREQ_MIN, Math.min(FREQ_MAX, freq)));
        const logMin = Math.log10(FREQ_MIN);
        const logMax = Math.log10(FREQ_MAX);
        return padding + ((logFreq - logMin) / (logMax - logMin)) * (width - 2 * padding);
    };

    const yScale = (db: number) => {
        const clampedDb = Math.max(DB_MIN, Math.min(DB_MAX, db));
        return height - padding - ((clampedDb - DB_MIN) / (DB_MAX - DB_MIN)) * (height - 2 * padding);
    };

    // Generate tick marks
    const generateFrequencyTicks = (): Array<{ x: number; label: string; freq: number }> => {
        const ticks: Array<{ x: number; label: string; freq: number }> = [];
        const frequencies = [20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000];

        frequencies.forEach(freq => {
            if (freq >= FREQ_MIN && freq <= FREQ_MAX) {
                const x = xScale(freq);
                const label = freq >= 1000 ? `${(freq / 1000).toFixed(0)}k` : freq.toString();
                ticks.push({ x, label, freq });
            }
        });

        return ticks;
    };

    const generateDbTicks = (): Array<{ y: number; label: string; db: number }> => {
        const ticks: Array<{ y: number; label: string; db: number }> = [];
        const dbValues = [-15, -10, -5, 0, 5, 10, 15];

        dbValues.forEach(db => {
            const y = yScale(db);
            ticks.push({ y, label: db.toString(), db });
        });

        return ticks;
    };

    // Prepare data for rendering
    const filteredData = data
        .filter(d => d.frequency >= FREQ_MIN && d.frequency <= FREQ_MAX && !isNaN(d.magnitude))
        .sort((a, b) => a.frequency - b.frequency);

    // Generate smooth curve path
    const generatePath = () => {
        if (filteredData.length === 0) return '';

        let path = `M ${xScale(filteredData[0].frequency)} ${yScale(filteredData[0].magnitude)}`;

        for (let i = 1; i < filteredData.length; i++) {
            path += ` L ${xScale(filteredData[i].frequency)} ${yScale(filteredData[i].magnitude)}`;
        }

        return path;
    };

    const freqTicks = generateFrequencyTicks();
    const dbTicks = generateDbTicks();
    
    return (
        <div className="w-full overflow-x-auto">
            <svg width={width} height={height} className="border border-racing-green/20 rounded-lg bg-white">
                {/* Grid lines */}
                <defs>
                    <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse">
                        <path d="M 20 0 L 0 0 0 20" fill="none" stroke="#004225" strokeWidth="0.3" opacity="0.05"/>
                    </pattern>
                </defs>
                <rect width="100%" height="100%" fill="url(#grid)" />

                {/* Grid lines */}
                {freqTicks.map(tick => (
                    <line
                        key={`v-grid-${tick.freq}`}
                        x1={tick.x}
                        y1={padding}
                        x2={tick.x}
                        y2={height - padding}
                        stroke="#004225"
                        strokeWidth="0.5"
                        opacity="0.1"
                    />
                ))}
                {dbTicks.map(tick => (
                    <line
                        key={`h-grid-${tick.db}`}
                        x1={padding}
                        y1={tick.y}
                        x2={width - padding}
                        y2={tick.y}
                        stroke="#004225"
                        strokeWidth="0.5"
                        opacity="0.1"
                    />
                ))}

                {/* 0dB reference line */}
                <line
                    x1={padding}
                    y1={yScale(0)}
                    x2={width - padding}
                    y2={yScale(0)}
                    stroke="#004225"
                    strokeWidth="1"
                    opacity="0.3"
                />

                {/* Axes */}
                <line x1={padding} y1={height - padding} x2={width - padding} y2={height - padding} stroke="#004225" strokeWidth="1"/>
                <line x1={padding} y1={padding} x2={padding} y2={height - padding} stroke="#004225" strokeWidth="1"/>

                {/* Data curve */}
                {filteredData.length > 0 && (
                    <path
                        d={generatePath()}
                        fill="none"
                        stroke="#b8860b"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                    />
                )}

                {/* Data points (subtle) */}
                {filteredData.map((d, i) => (
                    <circle
                        key={`point-${i}`}
                        cx={xScale(d.frequency)}
                        cy={yScale(d.magnitude)}
                        r="1.5"
                        fill="#004225"
                        opacity="0.6"
                    />
                ))}

                {/* Frequency tick marks and labels */}
                {freqTicks.map(tick => (
                    <g key={`freq-tick-${tick.freq}`}>
                        <line
                            x1={tick.x}
                            y1={height - padding}
                            x2={tick.x}
                            y2={height - padding + 5}
                    stroke="#004225"
                            strokeWidth="1"
                        />
                        <text
                            x={tick.x}
                            y={height - padding + 18}
                            textAnchor="middle"
                            className="text-xs fill-current text-racing-green font-medium"
                        >
                            {tick.label}
                        </text>
                    </g>
                ))}

                {/* dB tick marks and labels */}
                {dbTicks.map(tick => (
                    <g key={`db-tick-${tick.db}`}>
                        <line
                            x1={padding - 5}
                            y1={tick.y}
                            x2={padding}
                            y2={tick.y}
                            stroke="#004225"
                            strokeWidth="1"
                        />
                        <text
                            x={padding - 8}
                            y={tick.y + 4}
                            textAnchor="end"
                            className="text-xs fill-current text-racing-green font-medium"
                        >
                            {tick.label}
                        </text>
                    </g>
                ))}

                {/* Axis labels */}
                <text
                    x={width / 2}
                    y={height - 10}
                    textAnchor="middle"
                    className="text-sm fill-current text-racing-green font-semibold"
                >
                    Frequency (Hz)
                </text>
                <text
                    x={15}
                    y={height / 2}
                    textAnchor="middle"
                    transform={`rotate(-90 15 ${height / 2})`}
                    className="text-sm fill-current text-racing-green font-semibold"
                >
                    Magnitude (dB)
                </text>
            </svg>
        </div>
    );
};

export default FrequencyChart;
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

1. **WebM/OGG browser recording** - Convert to WAV for Python compatibility using ffmpeg
2. **Room dimensions enhance acoustics** - Optional user input enables resonance mode calculations
3. **Single processing screen** - Eliminated jarring upload/processing phase transitions
4. **Custom SVG frequency charts** - Professional dB scales with logarithmic frequency axes
5. **Always use pre-signed URLs for file uploads** - Never stream through the server
6. **FIFINE K669 calibration is hardcoded** - Can add more mic profiles later
7. **Polling every 2 seconds is sufficient** - WebSockets are overkill for this
8. **Use JSONB for frequency data** - Flexible and performant
9. **Repository pattern for all database access** - Clean separation of concerns
10. **Mock all external dependencies in tests** - Fast, reliable tests
11. **British Racing Green theme throughout** - Consistent visual identity
12. **Cache AI responses** - OpenAI API is expensive
13. **Progress updates are UX critical** - Users need feedback during processing
14. **Test coverage target is 80%** - Shows production quality
15. **Real-time frontend-backend sync** - Live status polling and seamless navigation