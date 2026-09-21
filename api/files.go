package api

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"echosystem/util"
)

// FileItem represents a single file or directory in the explorer
type FileItem struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// BaseDir is the root directory we are allowed to explore
var BaseDir string

func init() {
	// Resolve $HOME/public to an absolute path securely
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp" // Fallback if home dir is not found
	}
	BaseDir = filepath.Join(homeDir, "public")

	// Ensure the directory exists
	if errMkdir := os.MkdirAll(BaseDir, 0o755); errMkdir != nil {
		util.HandleFSError(errMkdir, BaseDir, "public directory creation")
		return
	}
}

// FileExplorerHandler handles requests to list directory contents
func FileExplorerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get requested path, default to root "/"
	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		requestedPath = "/"
	}

	// Security: Clean the path to prevent directory traversal (e.g., "../")
	cleanPath := filepath.Clean(requestedPath)

	// Join with base directory and get absolute path
	fullPath := filepath.Join(BaseDir, cleanPath)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Security: Verify the resolved path is still inside BaseDir
	absBaseDir, _ := filepath.Abs(BaseDir)
	if !strings.HasPrefix(absFullPath, absBaseDir) {
		http.Error(w, "forbidden: access denied", http.StatusForbidden)
		return
	}

	// Read directory contents
	entries, err := os.ReadDir(absFullPath)
	if err != nil {
		http.Error(w, "failed to read directory", http.StatusInternalServerError)
		return
	}

	var items []FileItem

	// Add "Go Back" option if not in root
	if cleanPath != "/" {
		items = append(items, FileItem{
			Name:  "..",
			IsDir: true,
			Size:  0,
		})
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, FileItem{
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	// sort items: directories first, then by file type, then alphabetically
	sort.Slice(items, func(i, j int) bool {
		// keep ".." always at the very top
		if items[i].Name == ".." {
			return true
		}
		if items[j].Name == ".." {
			return false
		}

		// directories come before files
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}

		// if both are files, group by extension first
		if !items[i].IsDir && !items[j].IsDir {
			extI := strings.ToLower(filepath.Ext(items[i].Name))
			extJ := strings.ToLower(filepath.Ext(items[j].Name))

			// different extensions: sort by extension
			if extI != extJ {
				return extI < extJ
			}
		}

		// same type or both directories: alphabetical order
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// =======================================================
// FILE SERVER HANDLING
// =======================================================

// PublicFileHandler serves public files with inline or attachment disposition
func PublicFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestedPath := strings.TrimPrefix(r.URL.Path, "/public/")
	cleanPath := filepath.Clean(requestedPath)
	fullPath := filepath.Join(BaseDir, cleanPath)

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// security: block paths outside the public directory
	absBaseDir, _ := filepath.Abs(BaseDir)
	if absFullPath != absBaseDir && !strings.HasPrefix(absFullPath, absBaseDir+string(os.PathSeparator)) {
		http.Error(w, "forbidden: access denied", http.StatusForbidden)
		return
	}

	info, err := os.Stat(absFullPath)
	if err != nil || info.IsDir() {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// security: hide dotfiles from public access
	if strings.HasPrefix(info.Name(), ".") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	ctype := detectContentType(absFullPath)

	if isViewable(ctype) {
		w.Header().Set("Content-Disposition", "inline")
	} else {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", info.Name()))
	}

	w.Header().Set("Content-Type", ctype)
	http.ServeFile(w, r, absFullPath)
}

// detectContentType reads the file header to guess its mime type
func detectContentType(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	ctype := http.DetectContentType(buf[:n])

	// fallback to extension when sniffing returns a generic type
	if ctype == "application/octet-stream" {
		if extType := mime.TypeByExtension(filepath.Ext(path)); extType != "" {
			ctype = extType
		}
	}
	return ctype
}

// isViewable reports whether browsers render this mime type natively
func isViewable(ctype string) bool {
	switch {
	case strings.HasPrefix(ctype, "image/"):
		return true
	case strings.HasPrefix(ctype, "text/"):
		return true
	case strings.HasPrefix(ctype, "video/"):
		return true
	case strings.HasPrefix(ctype, "audio/"):
		return true
	case ctype == "application/pdf":
		return true
	default:
		return false
	}
}
