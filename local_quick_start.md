# Sonara Easy Start Guide

## Prerequisites
- Docker and Docker Compose installed
- Go 1.21+ installed
- Node.js 18+ installed
- Make installed

## Quick Start Steps

1. **Start Docker services** - Launches PostgreSQL database, MinIO storage, and Python analyzer container
   ```bash
   docker-compose up -d
   ```

2. **Run database migrations** - Sets up the database schema and tables
   ```bash
   make migrate-up
   ```

3. **Start the backend API** - Launches the Go server on port 8080
   ```bash
   make run
   ```

4. **Start the frontend** - Launches the React development server on port 5173
   ```bash
   cd web && npm run dev
   ```

## Access the Application
- **Web App**: http://localhost:5173
- **API**: http://localhost:8080
- **MinIO Console**: http://localhost:9001 (admin/admin)

## Stop Everything
```bash
docker-compose down
Control + C to stop backend and frontend each
```
