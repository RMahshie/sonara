package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Service handles file storage operations
type S3Service interface {
	GenerateUploadURL(ctx context.Context, key string, contentType string) (string, error)
	GenerateDownloadURL(ctx context.Context, key string) (string, error)
	DownloadFile(ctx context.Context, key string) ([]byte, error)
	DeleteFile(ctx context.Context, key string) error
}

type s3Service struct {
	client    *s3.Client
	bucket    string
	urlExpiry time.Duration
	endpoint  string // For MinIO compatibility
}

// S3Config holds configuration for S3 service
type S3Config struct {
	Bucket    string
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
}

// NewS3Service creates a new S3 service instance
func NewS3Service(cfg S3Config) (S3Service, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET is required")
	}

	var awsCfg aws.Config
	var err error

	var client *s3.Client

	if cfg.Endpoint != "" {
		// MinIO configuration
		awsCfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion("us-east-1"), // MinIO doesn't care about region
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}

		endpoint := cfg.Endpoint
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = "http://" + endpoint
		}

		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = &endpoint
			o.UsePathStyle = true // MinIO requires path-style URLs
		})
	} else {
		// AWS S3 configuration
		awsCfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(cfg.Region),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}

		client = s3.NewFromConfig(awsCfg)
	}

	return &s3Service{
		client:    client,
		bucket:    cfg.Bucket,
		urlExpiry: 15 * time.Minute, // 15 minutes for uploads
		endpoint:  cfg.Endpoint,
	}, nil
}

// GenerateUploadURL generates a pre-signed URL for uploading files
func (s *s3Service) GenerateUploadURL(ctx context.Context, key string, contentType string) (string, error) {
	// Validate content type
	if err := s.validateContentType(contentType); err != nil {
		return "", err
	}

	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = s.urlExpiry
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return request.URL, nil
}

// GenerateDownloadURL generates a pre-signed URL for downloading files
func (s *s3Service) GenerateDownloadURL(ctx context.Context, key string) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 24 * time.Hour // Downloads valid for 24 hours
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return request.URL, nil
}

// DownloadFile downloads a file from S3/MinIO
func (s *s3Service) DownloadFile(ctx context.Context, key string) ([]byte, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	defer result.Body.Close()

	data := make([]byte, 0)
	buf := make([]byte, 1024)

	for {
		n, err := result.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
	}

	return data, nil
}

// DeleteFile deletes a file from S3/MinIO
func (s *s3Service) DeleteFile(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// validateContentType validates that the content type is supported
func (s *s3Service) validateContentType(contentType string) error {
	validTypes := map[string]bool{
		"audio/wav":  true,
		"audio/mpeg": true,
		"audio/flac": true,
		"audio/webm": true, // Browser MediaRecorder WebM format
		"audio/ogg":  true, // Browser MediaRecorder OGG format (fallback)
	}

	if !validTypes[contentType] {
		return fmt.Errorf("invalid content type: %s. Supported types: audio/wav, audio/mpeg, audio/flac, audio/webm, audio/ogg", contentType)
	}

	return nil
}
