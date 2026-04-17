package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/julienschmidt/httprouter"
)

// safePath ensures a path doesn't escape the base directory via traversal attacks
func safePath(base, userPath string) (string, error) {
	fullPath := filepath.Join(base, userPath)
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("invalid base path: %w", err)
	}
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	// Ensure the resolved path is within base directory
	if !isWithin(absPath, absBase) {
		return "", fmt.Errorf("path traversal detected")
	}
	return absPath, nil
}

// isWithin checks if a path is within a base directory
func isWithin(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && rel != ".." && !startsWith(rel, "..")
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func UploadImages(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	baseDir := "public/uploads"
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		http.Error(w, "Error preparing upload directory", http.StatusInternalServerError)
		return
	}

	var savedPaths []string
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		ext := filepath.Ext(fileHeader.Filename)
		// Generate safe filename based on timestamp (prevents user control)
		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)

		// Use safePath to prevent directory traversal
		fullPath, err := safePath(baseDir, filename)
		if err != nil {
			http.Error(w, "Invalid file path", http.StatusBadRequest)
			return
		}

		out, err := os.Create(fullPath)
		if err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			http.Error(w, "Error writing file", http.StatusInternalServerError)
			return
		}

		publicPath := "/uploads/" + filename
		savedPaths = append(savedPaths, publicPath)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{
		"imageUrls": savedPaths,
	})
}
