-- Create analysis_results and room_info tables

-- Analysis results (JSONB for flexibility)
CREATE TABLE analysis_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
    frequency_data JSONB NOT NULL,  -- Array of {frequency, magnitude}
    rt60 FLOAT,
    room_modes JSONB,  -- Array of frequencies
    metrics JSONB,     -- Additional metrics
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_analysis_id ON analysis_results(analysis_id);

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

