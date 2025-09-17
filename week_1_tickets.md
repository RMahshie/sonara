# Sonara - Complete Week 1 Tickets (Setup + Foundation)

## Setup Sprint (Weekend Before)

### SETUP-001: Initialize Development Environment
**Priority:** P0 | **Estimate:** 1h | **Labels:** setup, environment
**Purpose:** Install all development tools on your Mac M2 - without these, nothing else works.
**Key Tasks:** 
- Install Docker Desktop for containerized services
- Install Go 1.21+, Node 18+, golang-migrate
- Configure VS Code/Cursor with Go, TypeScript, and Tailwind extensions
- Verify all tool versions
**Effect:** Development machine ready with Docker for services, native tools for development

### SETUP-002: Create GitHub Repository with CI/CD
**Priority:** P0 | **Estimate:** 1h | **Labels:** setup, git, ci
**Purpose:** Professional version control with automated testing from day one
**Key Tasks:** 
- Create GitHub repo "sonara" with comprehensive .gitignore
- Set up folder structure (cmd/, internal/, web/, scripts/, migrations/)
- Configure GitHub Actions for automated Go tests on push
- Enable branch protection requiring PR reviews
**Effect:** Every push triggers tests, main branch protected, clean project structure established

### SETUP-003: Go API Scaffold with Huma + Chi
**Priority:** P0 | **Estimate:** 2h | **Labels:** backend, api, setup
**Purpose:** Build API skeleton with automatic OpenAPI documentation and validation
**Key Implementation:** 
- Initialize Go module with Huma v2 + Chi router
- Create main.go with health endpoint at /health
- Set up automatic OpenAPI generation
- Add Swagger UI at /docs
- Configure structured logging with zerolog
- Create Makefile with run/test/build commands
**Effect:** API running on :8080 with self-documenting endpoints and health check

### SETUP-004: React Frontend with TypeScript and Tailwind
**Priority:** P0 | **Estimate:** 2h | **Labels:** frontend, setup
**Purpose:** Modern frontend with type safety and British Racing Green theme baked in from day one
**Key Implementation:** 
- Create Vite + React + TypeScript app
- Install and configure Tailwind CSS
- Add custom color palette (#004225 racing-green, #b8860b brass, #fef3c7 cream)
- Import Playfair Display (headers) and Inter (body) fonts
- Install core dependencies (axios, recharts, zustand, react-router-dom)
- Configure Vitest for testing
**Effect:** Frontend running on :5173 with consistent British Racing Green theme applied

### SETUP-005: Configure Railway and AWS S3
**Priority:** P0 | **Estimate:** 1h | **Labels:** infrastructure, deployment
**Purpose:** Production infrastructure ready from the start - no scrambling later
**Key Tasks:** 
- Create Railway project with PostgreSQL addon
- Set up AWS S3 bucket "sonara-audio-files" with proper CORS (production only)
- Generate AWS IAM credentials with S3 access
- Configure Railway environment variables (DATABASE_URL, AWS keys, etc.)
- Test deployment with Railway CLI
**Effect:** Can deploy to production immediately, database provisioned, S3 ready for production

### SETUP-006: Configure Docker Services for Development
**Priority:** P0 | **Estimate:** 1h | **Labels:** setup, docker, database
**Purpose:** Set up containerized services for clean, isolated local development
**Key Tasks:**
- Create docker-compose.yml with PostgreSQL 15, MinIO (local S3), and Python analyzer
- Configure MinIO as local S3 replacement (free, fast, no internet needed)
- Build Python container with numpy/scipy/librosa for audio processing
- Add Makefile commands (db-up, db-down, db-reset, services-up)
- Test all services start correctly with `docker-compose up`
**Docker Services:**
```yaml
postgres: PostgreSQL 15 on port 5432
minio: S3-compatible storage on port 9000/9001
analyzer: Python 3.11 with audio libraries
```
**Effect:** All backend services running in Docker, consistent environment across team

---

## Additional Configuration Files Needed

### Docker Compose Configuration (docker-compose.yml)
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: sonara
      POSTGRES_PASSWORD: localdev
      POSTGRES_DB: sonara_dev
    ports: ["5432:5432"]
    volumes: [postgres_data:/var/lib/postgresql/data]
  
  minio:
    image: minio/minio
    ports: ["9000:9000", "9001:9001"]
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"
    volumes: [minio_data:/data]
  
  analyzer:
    build: ./scripts
    volumes: [./scripts:/app, /tmp:/tmp]
    command: tail -f /dev/null

volumes:
  postgres_data:
  minio_data:
```

### Environment Variables (.env.development)
```bash
# Database (Docker PostgreSQL)
DATABASE_URL=postgresql://sonara:localdev@localhost:5432/sonara_dev?sslmode=disable

# S3 (MinIO for local, AWS for production)
S3_ENDPOINT=http://localhost:9000
S3_BUCKET=sonara-audio
AWS_ACCESS_KEY_ID=minioadmin
AWS_SECRET_ACCESS_KEY=minioadmin

# Python analyzer in Docker
PYTHON_COMMAND=docker exec analyzer python /app/analyze_audio.py
```

### Makefile Commands
```makefile
# Start all Docker services
services-up:
	docker-compose up -d
	@echo "Services running: PostgreSQL(:5432), MinIO(:9000), Python analyzer"

# Stop all services
services-down:
	docker-compose down

# Database specific
db-reset:
	docker-compose down -v postgres
	docker-compose up -d postgres
	make migrate-up

# Run migrations
migrate-up:
	migrate -path migrations -database "${DATABASE_URL}" up

# Development servers (run natively)
run:
	go run cmd/server/main.go

run-web:
	cd web && pnpm dev
```

---

## Week 1: Foundation Sprint

### API-001: Create Core REST API Endpoints
**Priority:** P0 | **Estimate:** 2h | **Labels:** backend, api
**Purpose:** Define the contract between frontend and backend - every future feature builds on these three endpoints
**Implementation Details:**
- POST /api/analyses - Creates analysis record, generates pre-signed S3 URL, returns analysis ID
- GET /api/analyses/{id}/status - Returns current status (pending/processing/completed/failed) and progress (0-100)
- GET /api/analyses/{id}/results - Returns frequency data and measurements when complete
- Use Huma's struct tags for automatic validation (file size <20MB, valid mime types)
**Effect:** Frontend can initiate uploads, track progress, and retrieve results with validated inputs

### API-002: Write API Endpoint Tests
**Priority:** P0 | **Estimate:** 2h | **Labels:** backend, testing
**Purpose:** Ensure validation works and catch regressions early - demonstrates professional code quality
**Test Coverage:**
- Table-driven tests for all endpoints
- Valid input acceptance (WAV/MP3/FLAC files)
- File size limit enforcement (reject >20MB)
- Invalid mime type rejection
- Proper HTTP status codes (201 for create, 400 for errors)
**Effect:** 80% API coverage achieved, validation verified, tests serve as living documentation

### DB-001: Design PostgreSQL Schema
**Priority:** P0 | **Estimate:** 2h | **Labels:** database, backend
**Purpose:** Database is the source of truth - must handle concurrent analyses and store large frequency data efficiently
**Schema Design:**
```sql
analyses table: id (UUID), session_id, status, progress, audio_s3_key, timestamps
analysis_results table: id, analysis_id (FK), frequency_data (JSONB), rt60, room_modes
room_info table: id, analysis_id (FK), room_size, ceiling_height, floor_type, features (JSONB)
```
**Key Decisions:**
- JSONB for flexible frequency data storage
- Proper indexes on session_id, status, created_at
- UUID primary keys for security
**Database Migrations:** Use golang-migrate to version control schema changes (up/down migrations)
**Local Setup:** PostgreSQL runs in Docker container via docker-compose
**Effect:** Persistent storage ready with flexibility for frequency arrays and future expansion

### DB-002: Implement Repository Pattern
**Priority:** P0 | **Estimate:** 2h | **Labels:** backend, database, testing
**Purpose:** Clean separation between business logic and database - standard enterprise pattern for maintainability
**Implementation:**
- Define repository interfaces (Create, GetByID, UpdateStatus, StoreResults)
- PostgreSQL implementation using pgx driver
- Transaction support for consistency
- Use testcontainers-go for real PostgreSQL in tests (spins up temporary containers)
**Testing Strategy:** Tests use isolated PostgreSQL containers, no shared test database needed
**Local Development:** Connect to PostgreSQL running in Docker on localhost:5432
**Effect:** Mockable data layer, all SQL queries centralized, 90% test coverage on data access

### S3-001: Build S3 Service for File Management
**Priority:** P0 | **Estimate:** 2h | **Labels:** backend, storage
**Purpose:** Never stream large audio files through server - use pre-signed URLs like Spotify/SoundCloud
**Key Features:**
- GenerateUploadURL with 15-minute expiry
- File validation (only WAV/MP3/FLAC, max 20MB)
- GenerateDownloadURL for processing
- AWS SDK v2 integration
**Local Development:** Use MinIO (S3-compatible storage running in Docker on port 9000)
**Production:** Use real AWS S3
**Configuration:** Service detects environment and uses appropriate endpoint (MinIO local, S3 prod)
**Effect:** Direct browser-to-storage uploads preserve server bandwidth, works identically in dev/prod

### S3-002: Test S3 Service with Mocks
**Priority:** P1 | **Estimate:** 1h | **Labels:** backend, testing
**Purpose:** Test S3 operations without making actual AWS calls or incurring costs
**Test Scenarios:**
- Mock AWS S3 client
- Test pre-signed URL generation
- Validate content type checking
- Test error handling
**Effect:** Fast, reliable tests that run without network dependencies

### AUDIO-001: Python FFT Analysis Script in Docker
**Priority:** P0 | **Estimate:** 3h | **Labels:** audio, python, core, docker
**Purpose:** The heart of Sonara - performs actual frequency analysis with FIFINE K669 microphone calibration
**Implementation:**
- Create Python script with librosa for audio loading (WAV/MP3/FLAC)
- Apply Hamming window to reduce spectral leakage
- Perform FFT to get frequency content
- Apply FIFINE K669 calibration curve (compensates for mic response)
- Return JSON with ~1000 frequency/magnitude points
**Docker Setup:**
- Python runs in Docker container with consistent numpy/scipy/librosa versions
- Script location: /app/analyze_audio.py in container
- Shared /tmp directory for audio file access
**Calibration Curve:** +12dB at 20Hz, flat at 1kHz, -3dB at 8kHz, +5dB at 20kHz
**Effect:** Accurate frequency response in isolated Python environment

### AUDIO-002: Test Audio Analysis Accuracy
**Priority:** P1 | **Estimate:** 2h | **Labels:** audio, testing, python
**Purpose:** Verify FFT accuracy - users rely on measurement correctness
**Test Strategy:**
- Generate 1kHz test tone, verify peak detection within 1%
- Test calibration curve interpolation
- Verify multiple sample rates (44.1kHz, 48kHz, 96kHz)
- Test edge cases (very quiet, very loud signals)
**Effect:** Audio processing accuracy validated with synthetic test signals

### UI-001: File Upload Component with Progress
**Priority:** P0 | **Estimate:** 2h | **Labels:** frontend, ui
**Purpose:** User's first interaction - needs to be intuitive with drag-drop, validation, and clear feedback
**Features:**
- Drag-and-drop zone with react-dropzone
- File type/size validation before upload
- Real-time upload progress bar
- Direct upload to S3 using pre-signed URL
- British Racing Green styling with hover states
**Effect:** Professional upload experience that builds user confidence

### UI-002: Test Upload Component
**Priority:** P1 | **Estimate:** 1h | **Labels:** frontend, testing
**Purpose:** Upload is critical path - if it breaks, the entire application is unusable
**Test Coverage:**
- File acceptance (valid audio files)
- File rejection (wrong type, too large)
- Progress bar updates
- Error message display
- Mock axios for upload simulation
**Effect:** Upload reliability verified with React Testing Library

### PROC-001: Complete Processing Pipeline
**Priority:** P0 | **Estimate:** 3h | **Labels:** backend, processing, core
**Purpose:** Orchestrates the entire analysis flow - the conductor that makes everything work together
**Pipeline Steps:**
1. Download audio from S3/MinIO (progress: 20%)
2. Save to shared /tmp directory
3. Call Python via docker exec: `docker exec analyzer python /app/analyze_audio.py /tmp/audio.wav` (progress: 50%)
4. Parse JSON results (progress: 80%)
5. Store in PostgreSQL (progress: 90%)
6. Mark complete (progress: 100%)
**Docker Integration:** Python analyzer runs in container, Go coordinates via docker exec
**Error Handling:** Each step wrapped in error handling, failures update status to "failed"
**Effect:** Complete end-to-end analysis with containerized Python processing

### PROC-002: Test Pipeline with Mocks
**Priority:** P1 | **Estimate:** 2h | **Labels:** backend, testing
**Purpose:** Test orchestration logic without requiring S3, Python, or database
**Mock Strategy:**
- Mock S3 download
- Mock Python subprocess execution
- Mock database operations
- Test success path and failure scenarios
- Verify cleanup always happens
**Effect:** Pipeline reliability verified in <500ms tests

### POLL-001: Implement Progress Polling System
**Priority:** P0 | **Estimate:** 2h | **Labels:** frontend, backend
**Purpose:** Users need real-time feedback during the 10-30 second analysis process
**Implementation:**
- Backend status endpoint with progress percentage and human-readable message
- React hook polling every 2 seconds
- Automatic stop on completion or failure
- Progressive messages ("Downloading...", "Analyzing frequencies...", "Finalizing...")
**Effect:** Real-time progress updates without WebSocket complexity

### POLL-002: Test Polling System
**Priority:** P1 | **Estimate:** 1h | **Labels:** frontend, backend, testing
**Purpose:** Verify polling starts/stops correctly without memory leaks
**Test Coverage:**
- Backend status message generation
- React hook with Jest fake timers
- Verify polling stops on completion
- Test cleanup on component unmount
**Effect:** Polling reliability confirmed, no memory leaks

---

## Week 1 Deliverable Summary

### Working Features
✅ Upload audio files (WAV/MP3/FLAC, max 20MB)  
✅ Direct-to-MinIO upload locally, S3 in production  
✅ Python FFT analysis in Docker container with FIFINE K669 calibration  
✅ Results stored in PostgreSQL (Docker) with JSONB  
✅ 2-second polling for progress updates  
✅ Basic frequency data retrieval  

### Technical Foundation
✅ Hybrid Docker setup (services containerized, Go/React native for hot-reload)  
✅ MinIO for free local S3 development  
✅ PostgreSQL in Docker for clean database isolation  
✅ Python in Docker for consistent audio processing environment  
✅ Huma API with automatic OpenAPI documentation  
✅ Repository pattern for clean data access  
✅ 80%+ backend test coverage with testcontainers  
✅ British Racing Green theme consistently applied  

### Development Workflow
```bash
# Start your day
docker-compose up -d    # Start PostgreSQL, MinIO, Python analyzer
make migrate-up         # Run any new migrations
make run               # Start Go API (native)
pnpm dev               # Start React (native) in another terminal

# End of day
docker-compose down    # Stop all Docker services
```

### What's NOT Done Yet
❌ RT60 calculation (Week 2)  
❌ Room mode detection (Week 2)  
❌ Frequency response charts (Week 2)  
❌ PDF export (Week 2)  
❌ AI recommendations (Week 3)  
❌ Production Dockerfile (Week 4)  
❌ Polish and UX refinement (Week 4)  

By the end of Week 1, you have a functional audio analyzer with proper Docker isolation for services while maintaining fast development iteration for your application code.