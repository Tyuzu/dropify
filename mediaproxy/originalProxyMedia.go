package mediaproxy

import (
	"crypto/sha1"
	"encoding/hex"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "image/gif"
	_ "image/png"

	"github.com/disintegration/imaging"
	"github.com/julienschmidt/httprouter"
)

const (
	cacheDir         = "./cache/media"
	cacheMaxAge      = 72 * time.Hour
	clientTimeout    = 12 * time.Second
	maxPixelsAllowed = 4096 * 4096 // 16 million px bounding
)

// ProxyHandler fetches, caches, transforms, re-encodes and streams media
func ProxyHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	raw := strings.TrimPrefix(ps.ByName("url"), "/")

	var target string
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		target = raw
	} else {
		target = strings.Replace(raw, "/", "://", 1)
	}

	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	host := u.Hostname()
	if host == "localhost" ||
		host == "127.0.0.1" ||
		host == "::1" ||
		strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "172.") {
		http.Error(w, "blocked host", http.StatusForbidden)
		return
	}

	// ensure cache dir exists
	_ = os.MkdirAll(cacheDir, 0755)

	// include transformations in cache key
	cacheKey := u.String() + r.URL.RawQuery
	cachePath := filepath.Join(cacheDir, hashURL(cacheKey))

	// serve from cache
	if fi, err := os.Stat(cachePath); err == nil && time.Since(fi.ModTime()) < cacheMaxAge {
		f, _ := os.Open(cachePath)
		defer f.Close()
		http.ServeContent(w, r, "", fi.ModTime(), f)
		return
	}

	// fetch remote
	client := &http.Client{Timeout: clientTimeout}
	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("User-Agent", "MediaProxy/1.0")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, "remote error", http.StatusBadGateway)
		return
	}

	contentType := resp.Header.Get("Content-Type")
	isImage := strings.HasPrefix(contentType, "image/")

	// requested transform params
	wParam, _ := strconv.Atoi(r.URL.Query().Get("w"))
	hParam, _ := strconv.Atoi(r.URL.Query().Get("h"))
	qParam, _ := strconv.Atoi(r.URL.Query().Get("q"))
	if qParam <= 0 {
		qParam = 80
	}

	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "jpeg" // default format as requested
	}
	if format != "jpeg" && format != "webp" && format != "avif" {
		format = "jpeg"
	}

	// If remote is not image → stream directly with no decoding
	if !isImage {
		tmp := cachePath + ".tmp"
		out, err := os.Create(tmp)
		if err == nil {
			io.Copy(out, resp.Body)
			out.Close()
			os.Rename(tmp, cachePath)
		}
		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeFile(w, r, cachePath)
		return
	}

	// decode image
	srcImg, _, err := image.Decode(resp.Body)
	if err != nil {
		http.Error(w, "decode error", http.StatusBadGateway)
		return
	}

	// enforce pixel bounding
	totalPx := srcImg.Bounds().Dx() * srcImg.Bounds().Dy()
	if totalPx > maxPixelsAllowed {
		// serve original stream (no decode malfunction)
		tmp := cachePath + ".tmp"
		out, _ := os.Create(tmp)
		// recreate fresh fetch for streaming
		req2, _ := http.NewRequest("GET", u.String(), nil)
		res2, err := client.Do(req2)
		if err != nil {
			http.Error(w, "large stream fail", http.StatusBadGateway)
			return
		}
		io.Copy(out, res2.Body)
		out.Close()
		res2.Body.Close()
		os.Rename(tmp, cachePath)

		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeFile(w, r, cachePath)
		return
	}

	// resize if requested
	if wParam > 0 || hParam > 0 {
		srcImg = imaging.Resize(srcImg, wParam, hParam, imaging.Lanczos)
	}

	tmp := cachePath + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	// encode into selected format
	switch format {
	// case "webp":
	// 	webp.Encode(out, srcImg, &webp.Options{Quality: float32(qParam)})
	// 	w.Header().Set("Content-Type", "image/webp")

	// case "avif":
	// 	// imaging doesn't natively support AVIF; use jpeg fallback
	// 	jpeg.Encode(out, srcImg, &jpeg.Options{Quality: qParam})
	// 	w.Header().Set("Content-Type", "image/jpeg")

	default: // jpeg
		jpeg.Encode(out, srcImg, &jpeg.Options{Quality: qParam})
		w.Header().Set("Content-Type", "image/jpeg")
	}

	out.Close()
	os.Rename(tmp, cachePath)

	w.Header().Set("Cache-Control", "public, max-age=86400")

	f, _ := os.Open(cachePath)
	defer f.Close()
	io.Copy(w, f)
}

// hashURL → stable filename
func hashURL(u string) string {
	h := sha1.New()
	h.Write([]byte(u))
	return hex.EncodeToString(h.Sum(nil))
}

// package mediaproxy

// import (
// 	"crypto/sha1"
// 	"encoding/hex"
// 	"io"
// 	"net/http"
// 	"net/url"
// 	"os"
// 	"path/filepath"
// 	"strings"
// 	"time"

// 	"github.com/julienschmidt/httprouter"
// )

// const (
// 	cacheDir      = "./cache/media"  // directory to store cached files
// 	cacheMaxAge   = 24 * time.Hour   // duration before cache considered stale
// 	clientTimeout = 10 * time.Second // fetch timeout
// )

// func ProxyHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	raw := strings.TrimPrefix(ps.ByName("url"), "/")

// 	var target string
// 	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
// 		target = raw
// 	} else {
// 		target = strings.Replace(raw, "/", "://", 1)
// 	}

// 	u, err := url.Parse(target)
// 	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
// 		http.Error(w, "invalid url", http.StatusBadRequest)
// 		return
// 	}

// 	host := u.Hostname()
// 	if host == "localhost" ||
// 		host == "127.0.0.1" ||
// 		host == "::1" ||
// 		strings.HasPrefix(host, "10.") ||
// 		strings.HasPrefix(host, "192.168.") ||
// 		strings.HasPrefix(host, "172.") {
// 		http.Error(w, "blocked host", http.StatusForbidden)
// 		return
// 	}

// 	// ensure cache directory exists
// 	if err := os.MkdirAll(cacheDir, 0755); err != nil {
// 		http.Error(w, "cache dir error", http.StatusInternalServerError)
// 		return
// 	}

// 	// create a deterministic filename from the URL hash
// 	cachePath := filepath.Join(cacheDir, hashURL(u.String()))

// 	// if cached file exists and fresh, serve it directly
// 	if fi, err := os.Stat(cachePath); err == nil {
// 		if time.Since(fi.ModTime()) < cacheMaxAge {
// 			f, err := os.Open(cachePath)
// 			if err == nil {
// 				defer f.Close()
// 				http.ServeContent(w, r, "", fi.ModTime(), f)
// 				return
// 			}
// 		}
// 	}

// 	// fetch from network
// 	req, err := http.NewRequest("GET", u.String(), nil)
// 	if err != nil {
// 		http.Error(w, "failed request", http.StatusInternalServerError)
// 		return
// 	}
// 	req.Header.Set("User-Agent", "MediaProxy/1.0 (+https://indium.netlify.app)")
// 	req.Header.Set("Accept", "*/*")

// 	client := &http.Client{Timeout: clientTimeout}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		http.Error(w, "fetch failed", http.StatusBadGateway)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
// 		http.Error(w, "remote error", http.StatusBadGateway)
// 		return
// 	}

// 	// save response body to cache file
// 	tmpPath := cachePath + ".tmp"
// 	out, err := os.Create(tmpPath)
// 	if err == nil {
// 		_, copyErr := io.Copy(out, resp.Body)
// 		out.Close()
// 		if copyErr == nil {
// 			os.Rename(tmpPath, cachePath) // atomic replace
// 		} else {
// 			os.Remove(tmpPath)
// 		}
// 	} else {
// 		io.Copy(io.Discard, resp.Body)
// 	}

// 	// serve response to client
// 	for k, v := range resp.Header {
// 		if len(v) > 0 {
// 			w.Header().Set(k, v[0])
// 		}
// 	}
// 	w.Header().Set("Cache-Control", "public, max-age=86400")
// 	w.WriteHeader(resp.StatusCode)
// 	f, _ := os.Open(cachePath)
// 	if f != nil {
// 		defer f.Close()
// 		io.Copy(w, f)
// 	}
// }

// // hashURL creates a short filename-safe hash of the URL
// func hashURL(u string) string {
// 	h := sha1.New()
// 	h.Write([]byte(u))
// 	return hex.EncodeToString(h.Sum(nil))
// }
