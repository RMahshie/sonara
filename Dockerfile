# Production multi-stage Dockerfile
FROM golang:1.21 AS go-builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server cmd/server/main.go

FROM node:18 AS frontend-builder
WORKDIR /app
COPY web/package*.json ./
RUN npm ci
COPY web/ .
RUN npm run build

FROM python:3.11-slim
# Install Go binary
COPY --from=go-builder /app/server /usr/local/bin/server
# Install frontend
COPY --from=frontend-builder /app/dist /var/www/html
# Install Python deps
COPY scripts/requirements.txt .
RUN pip install -r requirements.txt
COPY scripts/*.py /app/

CMD ["server"]