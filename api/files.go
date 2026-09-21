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

	"echosystem/views"
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

// logTime formats the current timestamp for log prefixes
func logTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func init() {
	// resolve $HOME/public to an absolute path securely
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("[FATAL] (%s) no home directory found: %v\n", logTime(), err)
		os.Exit(1)
	}
	BaseDir = filepath.Join(homeDir, "public")
	fmt.Printf("[LOG] (%s) explorer: base directory resolved to: %s\n", logTime(), BaseDir)

	// verify the base directory exists and is readable at startup
	if info, err := os.Stat(BaseDir); err != nil {
		fmt.Printf("[WARN] (%s) explorer: base directory is NOT accessible: %v\n", logTime(), err)
	} else if !info.IsDir() {
		fmt.Printf("[WARN] (%s) explorer: base path exists but is not a directory: %s\n", logTime(), BaseDir)
	} else {
		fmt.Printf("[LOG] (%s) explorer: base directory exists and is readable\n", logTime())
	}
}

// FileExplorerHandler handles requests to list directory contents
func FileExplorerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fmt.Printf("[WARN] (%s) explorer: rejected method %s from %s\n", logTime(), r.Method, r.RemoteAddr)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// get requested path, default to root "/"
	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		requestedPath = "/"
	}
	fmt.Printf("[LOG] (%s) explorer: list requested by %s for path '%s'\n", logTime(), r.RemoteAddr, requestedPath)

	// security: clean the path to prevent directory traversal (e.g., "../")
	cleanPath := filepath.Clean(requestedPath)

	// join with base directory and get absolute path
	fullPath := filepath.Join(BaseDir, cleanPath)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		fmt.Printf("[ERROR] (%s) explorer: failed to resolve absolute path for '%s': %v\n", logTime(), requestedPath, err)
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	fmt.Printf("[LOG] (%s) explorer: resolved filesystem path: %s\n", logTime(), absFullPath)

	// security: verify the resolved path is still inside BaseDir
	absBaseDir, _ := filepath.Abs(BaseDir)
	if !strings.HasPrefix(absFullPath, absBaseDir) {
		fmt.Printf("[WARN] (%s) explorer: BLOCKED traversal attempt from %s targeting '%s'\n", logTime(), r.RemoteAddr, requestedPath)
		http.Error(w, "forbidden: access denied", http.StatusForbidden)
		return
	}

	// read directory contents
	entries, err := os.ReadDir(absFullPath)
	if err != nil {
		fmt.Printf("[ERROR] (%s) explorer: failed to read directory %s: %v\n", logTime(), absFullPath, err)
		http.Error(w, "failed to read directory", http.StatusInternalServerError)
		return
	}

	var items []FileItem

	// add "go back" option if not in root
	if cleanPath != "/" {
		items = append(items, FileItem{
			Name:  "..",
			IsDir: true,
			Size:  0,
		})
	}

	skipped := 0
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			skipped++
			fmt.Printf("[WARN] (%s) explorer: skipped unreadable entry '%s' in %s: %v\n", logTime(), entry.Name(), absFullPath, err)
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

	fmt.Printf("[LOG] (%s) explorer: listed %d items from %s (%d skipped)\n", logTime(), len(items), absFullPath, skipped)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// =======================================================
// FILE SERVER HANDLING
// =======================================================

// PublicFileHandler serves public files with inline or attachment disposition
func PublicFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		fmt.Printf("[WARN] (%s) public: rejected method %s from %s\n", logTime(), r.Method, r.RemoteAddr)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestedPath := strings.TrimPrefix(r.URL.Path, "/public/")
	fmt.Printf("[LOG] (%s) public: file requested by %s: '%s'\n", logTime(), r.RemoteAddr, requestedPath)

	cleanPath := filepath.Clean(requestedPath)
	fullPath := filepath.Join(BaseDir, cleanPath)

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		fmt.Printf("[ERROR] (%s) public: failed to resolve absolute path for '%s': %v\n", logTime(), requestedPath, err)
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	fmt.Printf("[LOG] (%s) public: resolved filesystem path: %s\n", logTime(), absFullPath)

	// security: block paths outside the public directory
	absBaseDir, _ := filepath.Abs(BaseDir)
	if absFullPath != absBaseDir && !strings.HasPrefix(absFullPath, absBaseDir+string(os.PathSeparator)) {
		fmt.Printf("[WARN] (%s) public: BLOCKED traversal attempt from %s targeting '%s'\n", logTime(), r.RemoteAddr, requestedPath)
		http.Error(w, "forbidden: access denied", http.StatusForbidden)
		return
	}

	info, err := os.Stat(absFullPath)
	if err != nil || info.IsDir() {
		if err != nil {
			fmt.Printf("[ERROR] (%s) public: stat failed for %s: %v\n", logTime(), absFullPath, err)
		} else {
			fmt.Printf("[WARN] (%s) public: refused to serve directory as file: %s\n", logTime(), absFullPath)
		}
		ctx := r.Context()
		err := views.NotAcceptable().Render(ctx, w)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	ctype := detectContentType(absFullPath)

	disposition := "inline"
	if isViewable(ctype) {
		w.Header().Set("Content-Disposition", "inline")
	} else {
		disposition = "attachment"
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", info.Name()))
	}

	fmt.Printf("[LOG] (%s) public: serving '%s' (%d bytes) as %s with disposition %s\n", logTime(), info.Name(), info.Size(), ctype, disposition)

	w.Header().Set("Content-Type", ctype)
	http.ServeFile(w, r, absFullPath)
}

// detectContentType reads the file header to guess its mime type
func detectContentType(path string) string {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("[ERROR] (%s) public: could not open %s for mime sniffing: %v\n", logTime(), path, err)
		return "application/octet-stream"
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	ctype := http.DetectContentType(buf[:n])

	// fallback to extension when sniffing returns a generic type
	if ctype == "application/octet-stream" {
		if extType := mime.TypeByExtension(filepath.Ext(path)); extType != "" {
			fmt.Printf("[LOG] (%s) public: sniffing was generic, using extension type %s for %s\n", logTime(), extType, filepath.Base(path))
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
