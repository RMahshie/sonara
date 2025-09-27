# Sonara Quickstart Guide

## 🧪 Running Tests

### Python Audio Analyzer Tests
```bash
# Install dependencies
cd scripts && source venv/bin/activate && pip install -r requirements.txt

# Run all analyzer tests
python3 test_analyze_audio.py
```

### Go API Tests

#### Unit Tests (Fast, No Docker Required)
```bash
# Run all unit tests
go test ./... -v -short
```

#### Integration Tests (Slow, Requires Docker)
```bash
# Run integration tests (spins up PostgreSQL + MinIO containers)
go test ./internal/processing -v

# Or run all tests including integration
go test ./... -v  # Without -short flag
```

### Web Frontend Tests
```bash
# Install dependencies
cd web && pnpm install

# Run tests
pnpm test
```

## 🚀 Local Development

### 0. Configure Python Command

**Development uses Docker containers for consistency:**

```bash
# Set PYTHON_CMD for development (uses analyzer container)
export PYTHON_CMD="docker exec analyzer python /app/analyze_audio.py"

# For production, this would be:
# export PYTHON_CMD="python /app/scripts/analyze_audio.py"
```

### 1. Start Infrastructure
```bash
# Start all Docker services (PostgreSQL, MinIO, Python analyzer)
make services-up

# Services will be available at:
# - PostgreSQL: localhost:5432
# - MinIO: localhost:9000 (web console), localhost:9001 (API)
# - Python analyzer: container ready for processing
```

### 2. Start Go API Server
```bash
# In a new terminal
make run

# API will be available at http://localhost:8080
```

### 3. Start Web Frontend
```bash
# In a new terminal
cd web && pnpm dev

# Frontend will be available at http://localhost:5173
```

### 4. Verify Everything Works
1. Open http://localhost:5173 in your browser
2. Upload an audio file
3. Check that analysis completes successfully
4. View frequency analysis results

## 🐳 Docker Deployment

### Build and Run Everything
```bash
# Build all containers
docker-compose build

# Start all services
docker-compose up -d

# Services available at:
# - Web: http://localhost:80
# - API: http://localhost:8080
# - MinIO Console: http://localhost:9001
```

### Individual Services
```bash
# Just the database and storage
docker-compose up -d postgres minio

# Just the API
docker-compose up -d api

# Just the web frontend
docker-compose up -d web
```

## 🔧 Development Workflow

### Daily Development
```bash
# Start infrastructure once
make services-up

# Terminal 1: API server (hot reload)
make dev

# Terminal 2: Web frontend (hot reload)
cd web && pnpm dev

# Terminal 3: Run tests as needed
go test ./... -short    # Unit tests
python3 scripts/test_analyze_audio.py  # Audio tests
```

### Database Management
```bash
# Reset database and run migrations
make db-reset

# Just run migrations
make migrate-up
```

### Clean Up
```bash
# Stop all services
make services-down

# Clean build artifacts
make clean
```

## 📊 Test Coverage

- **Unit Tests**: Core logic without external dependencies
- **Integration Tests**: Full pipeline with real database and file storage
- **Audio Tests**: FFT accuracy and calibration validation

Run `go test ./... -cover` for coverage reports.
