package config

import (
	"os"
	"strconv"
	"time"
)

// Upload configuration
var (
	// MaxUploadSize is the maximum size for single file uploads (default: 50MB)
	MaxUploadSize = getEnvInt64("MAX_UPLOAD_SIZE", 50*1024*1024)

	// ChunkUploadSize is the maximum size for chunked uploads (default: 50MB)
	ChunkUploadSize = getEnvInt64("CHUNK_UPLOAD_SIZE", 50*1024*1024)

	// ChunkBuffer is the buffer size for reading chunks (default: 256KB)
	ChunkBuffer = getEnvInt64("CHUNK_BUFFER", 1024*256)

	// TempUploadDir is the directory for temporary uploads (default: ./uploads/tmp)
	TempUploadDir = getEnv("TEMP_UPLOAD_DIR", "./uploads/tmp")

	// UploadDir is the base directory for uploads (default: ./uploads)
	UploadDir = getEnv("UPLOAD_DIR", "./uploads")

	// StaticUploadDir is the static assets upload directory (default: ./static/uploads)
	StaticUploadDir = getEnv("STATIC_UPLOAD_DIR", "./static/uploads")
)

// Cleanup configuration
var (
	// CleanupAge is the age after which temporary uploads are cleaned up (default: 2 minutes)
	CleanupAge = getEnvDuration("CLEANUP_AGE", 2*time.Minute)
)

// Rate Limiter configuration
var (
	// RateLimitRate is the rate limit (requests per second) - default: 1
	RateLimitRate = getEnvInt("RATE_LIMIT_RATE", 1)

	// RateLimitBurst is the burst size for rate limiting - default: 6
	RateLimitBurst = getEnvInt("RATE_LIMIT_BURST", 6)

	// RateLimitCleanup is the TTL for rate limiter entries - default: 10 minutes
	RateLimitCleanup = getEnvDuration("RATE_LIMIT_CLEANUP", 10*time.Minute)

	// RateLimitMaxEntries is the max concurrent IPs to track - default: 10000
	RateLimitMaxEntries = getEnvInt("RATE_LIMIT_MAX_ENTRIES", 10000)
)

// Virus scanning configuration
var (
	// VirusScanReadLimit is the amount of file to scan for malicious signatures (default: 1MB)
	VirusScanReadLimit = getEnvInt64("VIRUS_SCAN_READ_LIMIT", 1<<20)

	// MaxAllowedSizeScan is the maximum file size to attempt scanning (default: 1GB)
	MaxAllowedSizeScan = getEnvInt64("MAX_ALLOWED_SIZE_SCAN", 1<<30)
)

// Image processing configuration
var (
	// ImageQuality is the JPEG encoding quality (default: 90)
	ImageQuality = getEnvInt("IMAGE_QUALITY", 90)

	// MaxImageWidth is the maximum allowed image width (default: 8000)
	MaxImageWidth = getEnvInt("MAX_IMAGE_WIDTH", 8000)

	// MaxImageHeight is the maximum allowed image height (default: 8000)
	MaxImageHeight = getEnvInt("MAX_IMAGE_HEIGHT", 8000)
)

// Video processing configuration
var (
	// VideoMaxParallelProcesses is the number of video resolutions to process in parallel (default: 3)
	VideoMaxParallelProcesses = getEnvInt("VIDEO_MAX_PARALLEL_PROCESSES", 3)
)

// Security configuration
var (
	// DirectoryPermissions is the file mode for creating directories (default: 0700)
	DirectoryPermissions = os.FileMode(0700)

	// FilePermissions is the file mode for creating files (default: 0600)
	FilePermissions = os.FileMode(0600)
)

// Helper functions to read environment variables with defaults

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
