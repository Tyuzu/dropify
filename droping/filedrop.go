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
	"net"
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

const (
	maxUploadBytes = 200 << 20 // request cap (200MB)
	maxFileSize    = 50 << 20  // per file cap (50MB)

	baseUploadPath = "./static/uploads"

	maxImagePixels = 25_000_000
	maxWidth       = 8000
	maxHeight      = 8000
)

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

	// Remote upload
	if remoteURL := strings.TrimSpace(r.FormValue("remoteUrl")); remoteURL != "" {
		key := strings.ToLower(strings.TrimSpace(r.FormValue("remoteKey")))

		att, err := handleRemoteUpload(remoteURL, key, entityType, entityId)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, []Attachment{att})
		return
	}

	// Local upload
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

// ---------------- Core Save ----------------

func saveFile(reader io.Reader, mimeType, key, entityType, entityId string) (Attachment, error) {
	limited := &io.LimitedReader{
		R: reader,
		N: maxFileSize + 1,
	}

	ext := ".bin"
	if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
		ext = exts[0]
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	folder := resolveFolder(key)

	saveDir := filepath.Join(baseUploadPath, entityType, folder)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return Attachment{}, err
	}

	savePath := filepath.Join(saveDir, filename)

	out, err := os.Create(savePath)
	if err != nil {
		return Attachment{}, err
	}
	defer out.Close()

	written, err := io.Copy(out, limited)
	if err != nil {
		return Attachment{}, err
	}

	if written > maxFileSize {
		os.Remove(savePath)
		return Attachment{}, fmt.Errorf("file too large")
	}

	res := safeDetectResolution(savePath, mimeType)

	return Attachment{
		Filename:    filename,
		Extn:        ext,
		Key:         key,
		EntityType:  entityType,
		EntityID:    entityId,
		Resolutions: res,
	}, nil
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

	ext := strings.ToLower(filepath.Ext(fh.Filename))

	mimeType, err := validateFile(head, ext)
	if err != nil {
		return Attachment{}, err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Attachment{}, fmt.Errorf("file seek failed")
	}

	att, err := saveFile(file, mimeType, key, entityType, entityId)
	if err != nil {
		return Attachment{}, err
	}

	log.Printf("uploaded local: %s", att.Filename)
	return att, nil
}

// ---------------- Remote Upload ----------------

func handleRemoteUpload(remoteURL, key, entityType, entityId string) (Attachment, error) {
	u, err := url.Parse(remoteURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Attachment{}, fmt.Errorf("invalid url")
	}

	if isPrivateHost(u.Hostname()) {
		return Attachment{}, fmt.Errorf("blocked host")
	}

	client := &http.Client{
		Timeout: 12 * time.Second,
		Transport: &http.Transport{
			ResponseHeaderTimeout: 5 * time.Second,
		},
	}

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

	mimeType, err := validateFile(head, "")
	if err != nil {
		return Attachment{}, err
	}

	reader := io.MultiReader(bytes.NewReader(head), resp.Body)

	att, err := saveFile(reader, mimeType, key, entityType, entityId)
	if err != nil {
		return Attachment{}, err
	}

	log.Printf("uploaded remote: %s", att.Filename)
	return att, nil
}

// ---------------- Validation ----------------

func validateFile(head []byte, ext string) (string, error) {
	mimeType := http.DetectContentType(head)

	if !allowedMimeTypes[mimeType] {
		return "", fmt.Errorf("invalid mime type")
	}

	if ext != "" && !allowedExtensions[ext] {
		return "", fmt.Errorf("invalid extension")
	}

	if ext != "" {
		if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
			valid := false
			for _, e := range exts {
				if e == ext {
					valid = true
					break
				}
			}
			if !valid {
				return "", fmt.Errorf("mime/extension mismatch")
			}
		}
	}

	return mimeType, nil
}

// ---------------- Image Safety ----------------

func safeDetectResolution(path, mimeType string) []int {
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

	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil
	}

	if cfg.Width > maxWidth || cfg.Height > maxHeight {
		return nil
	}

	if cfg.Width*cfg.Height > maxImagePixels {
		return nil
	}

	return []int{cfg.Width, cfg.Height}
}

// ---------------- Helpers ----------------

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

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "..", "")
	s = strings.ReplaceAll(s, "/", "")
	return s
}

func isPrivateHost(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return true
	}

	privateRanges := []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"::1/128",
	}

	for _, cidr := range privateRanges {
		_, block, _ := net.ParseCIDR(cidr)
		if block.Contains(ip) {
			return true
		}
	}

	return false
}
