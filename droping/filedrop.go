package droping

import (
	"bytes"
	"dropify/utils"
	"fmt"
	"image"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/julienschmidt/httprouter"
)

const maxUploadBytes = 200 << 20 // 200MB

// ---------------- Allowed Types ----------------

var allowedMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,

	"audio/mpeg": true,
	"audio/wav":  true,
	"audio/ogg":  true,
}

var allowedExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".mp3": true, ".wav": true, ".ogg": true,
}

// ---------------- Folder Mapping ----------------

func resolveFolder(key string) string {
	switch key {
	case "banner":
		return "banner"
	case "photo":
		return "photo"
	case "poster":
		return "poster"
	case "thumb":
		return "thumb"
	case "image":
		return "images"
	case "audio":
		return "audio"
	case "video":
		return "videos"
	case "document":
		return "docs"
	case "gallery":
		return "gallery"
	default:
		return "misc"
	}
}

// ---------------- Handler ----------------

func FiledropHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		utils.RespondWithError(w, http.StatusBadRequest, "content-type must be multipart")
		return
	}

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	entityType := sanitize(r.FormValue("entityType"))
	entityId := sanitize(r.FormValue("entityId"))

	if entityType == "" {
		entityType = "misc"
	}

	// ---------------- Remote URL ingestion ----------------
	remoteURL := strings.TrimSpace(r.FormValue("remoteUrl"))
	remoteKey := strings.ToLower(strings.TrimSpace(r.FormValue("remoteKey")))

	if remoteURL != "" {
		att, err := handleRemoteUpload(remoteURL, remoteKey, entityType, entityId)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.RespondWithJSON(w, http.StatusOK, []Attachment{att})
		return
	}

	// ---------------- Normal Upload ----------------
	if r.MultipartForm == nil || len(r.MultipartForm.File) == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "no files provided")
		return
	}

	atts, err := processUploadedFiles(r, entityType, entityId)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, atts)
}

func OptionsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.WriteHeader(http.StatusNoContent)
}

// ---------------- Model ----------------

type Attachment struct {
	Filename    string `json:"filename"`
	Extn        string `json:"extn"`
	Key         string `json:"key"`
	EntityType  string `json:"entityType,omitempty"`
	EntityID    string `json:"entityId,omitempty"`
	Resolutions []int  `json:"resolutions,omitempty"`
}

// ---------------- Local Upload ----------------

func processUploadedFiles(r *http.Request, entityType, entityId string) ([]Attachment, error) {
	var attachments []Attachment

	for key, files := range r.MultipartForm.File {
		keyLower := strings.ToLower(key)

		for _, fh := range files {
			att, err := handleRegularUpload(fh, keyLower, entityType, entityId)
			if err != nil {
				return nil, err
			}
			attachments = append(attachments, att)
		}
	}

	return attachments, nil
}

func handleRegularUpload(fh *multipart.FileHeader, key, entityType, entityId string) (Attachment, error) {
	file, err := fh.Open()
	if err != nil {
		return Attachment{}, fmt.Errorf("cannot open file")
	}
	defer file.Close()

	head := make([]byte, 512)
	n, _ := file.Read(head)
	head = head[:n]

	mimeType := http.DetectContentType(head)

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext == "" {
		if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
			ext = exts[0]
		} else {
			ext = ".bin"
		}
	}

	if !allowedMimeTypes[mimeType] || !allowedExtensions[ext] {
		return Attachment{}, fmt.Errorf("file type not allowed")
	}

	file.Seek(0, io.SeekStart)

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	folder := resolveFolder(key)

	saveDir := filepath.Join("./uploads", entityType, folder)
	os.MkdirAll(saveDir, 0755)

	savePath := filepath.Join(saveDir, filename)

	out, err := os.Create(savePath)
	if err != nil {
		return Attachment{}, err
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return Attachment{}, err
	}

	res := detectResolution(savePath, mimeType)

	log.Printf("uploaded local: %s", filename)

	return Attachment{
		Filename:    filename,
		Extn:        ext,
		Key:         key,
		EntityType:  entityType,
		EntityID:    entityId,
		Resolutions: res,
	}, nil
}

// ---------------- Remote Upload (KEY PART) ----------------

func handleRemoteUpload(remoteURL, key, entityType, entityId string) (Attachment, error) {
	u, err := url.Parse(remoteURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Attachment{}, fmt.Errorf("invalid url")
	}

	if isPrivateHost(u.Hostname()) {
		return Attachment{}, fmt.Errorf("blocked host")
	}

	client := &http.Client{Timeout: 12 * time.Second}

	resp, err := client.Get(remoteURL)
	if err != nil {
		return Attachment{}, fmt.Errorf("fetch failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Attachment{}, fmt.Errorf("remote error")
	}

	head := make([]byte, 512)
	n, _ := resp.Body.Read(head)
	head = head[:n]

	mimeType := http.DetectContentType(head)

	if !allowedMimeTypes[mimeType] {
		return Attachment{}, fmt.Errorf("unsupported file type")
	}

	ext := ".jpg"
	if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
		ext = exts[0]
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	folder := resolveFolder(key)

	saveDir := filepath.Join("./uploads", entityType, folder)
	os.MkdirAll(saveDir, 0755)

	savePath := filepath.Join(saveDir, filename)

	out, err := os.Create(savePath)
	if err != nil {
		return Attachment{}, err
	}
	defer out.Close()

	reader := io.MultiReader(bytes.NewReader(head), resp.Body)

	if _, err := io.Copy(out, reader); err != nil {
		return Attachment{}, err
	}

	res := detectResolution(savePath, mimeType)

	log.Printf("uploaded remote: %s", filename)

	return Attachment{
		Filename:    filename,
		Extn:        ext,
		Key:         key,
		EntityType:  entityType,
		EntityID:    entityId,
		Resolutions: res,
	}, nil
}

// ---------------- Helpers ----------------

func detectResolution(path, mimeType string) []int {
	if !strings.HasPrefix(mimeType, "image/") {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return nil
	}

	return []int{cfg.Width, cfg.Height}
}

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "..", "")
	s = strings.ReplaceAll(s, "/", "")
	return s
}

func isPrivateHost(host string) bool {
	host = strings.ToLower(host)

	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	if strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "172.") {
		return true
	}

	return false
}
