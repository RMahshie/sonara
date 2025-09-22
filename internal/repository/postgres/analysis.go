package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/RMahshie/sonara/internal/repository"
	"github.com/RMahshie/sonara/pkg/models"
)

// PostgresAnalysisRepository implements AnalysisRepository for PostgreSQL
type PostgresAnalysisRepository struct {
	db *sql.DB
}

// NewPostgresAnalysisRepository creates a new PostgreSQL analysis repository
func NewPostgresAnalysisRepository(db *sql.DB) repository.AnalysisRepository {
	return &PostgresAnalysisRepository{db: db}
}

// Create inserts a new analysis record
func (r *PostgresAnalysisRepository) Create(ctx context.Context, analysis *models.Analysis) error {
	query := `
		INSERT INTO analyses (id, session_id, status, progress, audio_s3_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, query,
		analysis.ID,
		analysis.SessionID,
		analysis.Status,
		analysis.Progress,
		analysis.AudioS3Key,
		analysis.CreatedAt,
		analysis.UpdatedAt)

	return err
}

// GetByID retrieves an analysis by ID
func (r *PostgresAnalysisRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Analysis, error) {
	query := `
		SELECT id, session_id, status, progress, audio_s3_key, error_message, created_at, updated_at, completed_at
		FROM analyses
		WHERE id = $1`

	var analysis models.Analysis
	var audioS3Key, errorMsg sql.NullString
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&analysis.ID,
		&analysis.SessionID,
		&analysis.Status,
		&analysis.Progress,
		&audioS3Key,
		&errorMsg,
		&analysis.CreatedAt,
		&analysis.UpdatedAt,
		&completedAt)

	if err != nil {
		return nil, err
	}

	if audioS3Key.Valid {
		analysis.AudioS3Key = &audioS3Key.String
	}
	if errorMsg.Valid {
		analysis.ErrorMsg = &errorMsg.String
	}
	if completedAt.Valid {
		analysis.CompletedAt = &completedAt.Time
	}

	return &analysis, nil
}

// GetBySessionID retrieves analyses by session ID
func (r *PostgresAnalysisRepository) GetBySessionID(ctx context.Context, sessionID string) ([]*models.Analysis, error) {
	query := `
		SELECT id, session_id, status, progress, audio_s3_key, error_message, created_at, updated_at, completed_at
		FROM analyses
		WHERE session_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analyses []*models.Analysis
	for rows.Next() {
		var analysis models.Analysis
		var audioS3Key, errorMsg sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(
			&analysis.ID,
			&analysis.SessionID,
			&analysis.Status,
			&analysis.Progress,
			&audioS3Key,
			&errorMsg,
			&analysis.CreatedAt,
			&analysis.UpdatedAt,
			&completedAt)

		if err != nil {
			return nil, err
		}

		if audioS3Key.Valid {
			analysis.AudioS3Key = &audioS3Key.String
		}
		if errorMsg.Valid {
			analysis.ErrorMsg = &errorMsg.String
		}
		if completedAt.Valid {
			analysis.CompletedAt = &completedAt.Time
		}

		analyses = append(analyses, &analysis)
	}

	return analyses, nil
}

// UpdateStatus updates the status and progress of an analysis
func (r *PostgresAnalysisRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, progress int) error {
	query := `
		UPDATE analyses
		SET status = $1, progress = $2, updated_at = NOW(),
		    completed_at = CASE WHEN $1 = 'completed' THEN NOW() ELSE completed_at END
		WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, status, progress, id)
	return err
}

// UpdateError updates the error message for an analysis
func (r *PostgresAnalysisRepository) UpdateError(ctx context.Context, id uuid.UUID, errorMsg string) error {
	query := `
		UPDATE analyses
		SET status = 'failed', error_message = $1, updated_at = NOW()
		WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, errorMsg, id)
	return err
}

// StoreResults stores analysis results
func (r *PostgresAnalysisRepository) StoreResults(ctx context.Context, results *models.AnalysisResults) error {
	// Convert frequency data to JSON
	freqData, err := json.Marshal(results.FrequencyData)
	if err != nil {
		return fmt.Errorf("failed to marshal frequency data: %w", err)
	}

	// Convert room modes to JSON
	roomModes, err := json.Marshal(results.RoomModes)
	if err != nil {
		return fmt.Errorf("failed to marshal room modes: %w", err)
	}

	// Convert metrics to JSON
	var metrics []byte
	if results.Metrics != nil {
		metrics, err = json.Marshal(results.Metrics)
		if err != nil {
			return fmt.Errorf("failed to marshal metrics: %w", err)
		}
	}

	query := `
		INSERT INTO analysis_results (id, analysis_id, frequency_data, rt60, room_modes, metrics, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = r.db.ExecContext(ctx, query,
		results.ID,
		results.AnalysisID,
		string(freqData),
		results.RT60,
		string(roomModes),
		string(metrics),
		results.CreatedAt)

	return err
}

// GetResults retrieves analysis results
func (r *PostgresAnalysisRepository) GetResults(ctx context.Context, analysisID uuid.UUID) (*models.AnalysisResults, error) {
	query := `
		SELECT id, analysis_id, frequency_data, rt60, room_modes, metrics, created_at
		FROM analysis_results
		WHERE analysis_id = $1`

	var results models.AnalysisResults
	var rt60 sql.NullFloat64
	var roomModesStr, metricsStr sql.NullString

	err := r.db.QueryRowContext(ctx, query, analysisID).Scan(
		&results.ID,
		&results.AnalysisID,
		&results.FrequencyData, // This will need proper JSON unmarshaling
		&rt60,
		&roomModesStr,
		&metricsStr,
		&results.CreatedAt)

	if err != nil {
		return nil, err
	}

	if rt60.Valid {
		results.RT60 = &rt60.Float64
	}
	if roomModesStr.Valid {
		var roomModes []float64
		if err := json.Unmarshal([]byte(roomModesStr.String), &roomModes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal room modes: %w", err)
		}
		results.RoomModes = roomModes
	}
	if metricsStr.Valid {
		var metrics map[string]interface{}
		if err := json.Unmarshal([]byte(metricsStr.String), &metrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
		}
		results.Metrics = metrics
	}

	return &results, nil
}

// CreateRoomInfo inserts room information
func (r *PostgresAnalysisRepository) CreateRoomInfo(ctx context.Context, info *models.RoomInfo) error {
	features, err := json.Marshal(info.Features)
	if err != nil {
		return fmt.Errorf("failed to marshal features: %w", err)
	}

	query := `
		INSERT INTO room_info (id, analysis_id, room_size, ceiling_height, floor_type, features, speaker_placement, additional_notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = r.db.ExecContext(ctx, query,
		info.ID,
		info.AnalysisID,
		info.RoomSize,
		info.CeilingHeight,
		info.FloorType,
		string(features),
		info.SpeakerPlacement,
		info.AdditionalNotes,
		info.CreatedAt)

	return err
}

// GetRoomInfo retrieves room information by analysis ID
func (r *PostgresAnalysisRepository) GetRoomInfo(ctx context.Context, analysisID uuid.UUID) (*models.RoomInfo, error) {
	query := `
		SELECT id, analysis_id, room_size, ceiling_height, floor_type, features, speaker_placement, additional_notes, created_at
		FROM room_info
		WHERE analysis_id = $1`

	var info models.RoomInfo
	var featuresStr sql.NullString

	err := r.db.QueryRowContext(ctx, query, analysisID).Scan(
		&info.ID,
		&info.AnalysisID,
		&info.RoomSize,
		&info.CeilingHeight,
		&info.FloorType,
		&featuresStr,
		&info.SpeakerPlacement,
		&info.AdditionalNotes,
		&info.CreatedAt)

	if err != nil {
		return nil, err
	}

	if featuresStr.Valid {
		var features []string
		if err := json.Unmarshal([]byte(featuresStr.String), &features); err != nil {
			return nil, fmt.Errorf("failed to unmarshal features: %w", err)
		}
		info.Features = features
	}

	return &info, nil
}
