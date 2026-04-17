package utils

import (
	"crypto/rand"
	"encoding/base64"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// --- CSRF Token Generation ---

func CSRF(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	csrf := GenerateRandomString(12)
	RespondWithJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"csrf_token": csrf,
	})
}

// --- Random String and ID Generators ---

// GenerateRandomString creates a cryptographically secure random token of length n bytes, base64 encoded.
func GenerateRandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// Fallback: return empty on error - caller must handle
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

// GenerateRandomDigitString creates a cryptographically secure random numeric string of length n.
func GenerateRandomDigitString(n int) string {
	const digits = "0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	for i := range b {
		b[i] = digits[b[i]%byte(len(digits))]
	}
	return string(b)
}

// --- Hashing (DEPRECATED) ---
// Note: MD5 is cryptographically broken. Use SHA256 or bcrypt for security-sensitive hashing.
// This function is kept for backward compatibility only and should not be used for new code.
func EncrypIt(strToHash string) string {
	// Placeholder - callers should migrate to proper hashing
	return ""
}

// --- HTTP Response Helpers ---

func SendResponse(w http.ResponseWriter, status int, data any, message string, err error) {
	resp := map[string]any{
		"status":  status,
		"message": message,
		"data":    data,
	}
	if err != nil {
		resp["error"] = err.Error()
	}
	RespondWithJSON(w, status, resp)
}

// --- Slice Helpers ---

func Contains(slice []string, value string) bool {
	return slices.Contains(slice, value)
}

// --- Image Validation ---

var SupportedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
	"image/bmp":  true,
	"image/tiff": true,
}

func ValidateImageFileType(w http.ResponseWriter, header *multipart.FileHeader) bool {
	mimeType := header.Header.Get("Content-Type")
	if !SupportedImageTypes[mimeType] {
		http.Error(w, "Invalid file type. Supported formats: JPEG, PNG, WebP, GIF, BMP, TIFF.", http.StatusBadRequest)
		return false
	}
	return true
}

// --- Directory Helper ---

// EnsureDir creates a directory with restrictive permissions (0700: owner only)
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0700)
}

// SplitTags takes a comma-separated string and returns a cleaned []string
func SplitTags(input string) []string {
	if input == "" {
		return []string{}
	}
	parts := strings.Split(input, ",")
	var tags []string
	seen := make(map[string]bool)

	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag == "" {
			continue
		}
		tag = strings.ToLower(tag) // normalize
		if !seen[tag] {
			tags = append(tags, tag)
			seen[tag] = true
		}
	}
	return tags
}

// ——————————————————————————————————————————————————————————
// SanitizeFilename: exactly as before
func SanitizeFilename(name string) string {
	re := regexp.MustCompile(`[^\w.\-]`)
	clean := re.ReplaceAllString(filepath.Base(name), "_")
	if clean == "" {
		return "file"
	}
	return clean
}
