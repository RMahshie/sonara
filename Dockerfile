# Production multi-stage Dockerfile
FROM golang:1.24 AS builder
ARG CACHEBUST=1
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server cmd/server/main.go

FROM python:3.11-slim
WORKDIR /app

# Install ffmpeg for audio conversion
RUN apt-get update && apt-get install -y ffmpeg && rm -rf /var/lib/apt/lists/*

# Install uv package manager
RUN pip install uv

COPY --from=builder /app/server /usr/local/bin/server
COPY scripts/requirements.txt .
RUN uv pip install --system -r requirements.txt
COPY scripts/ ./scripts/
EXPOSE 8080
CMD ["server"]