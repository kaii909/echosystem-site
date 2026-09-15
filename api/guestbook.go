package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"echosystem/util"
)

type GuestbookEntry struct {
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
}

var (
	guestbook     []GuestbookEntry
	guestbookLock sync.Mutex
	GuestbookDir  = "/var/lib/echosystem/"
	guestbookFile = filepath.Join("/var/lib/echosystem", "guestbook.json")
)

// runs on package import
func init() {
	loadGuestbook()
}

// Loads entries from disk on startup
func loadGuestbook() {
	data, err := os.ReadFile(guestbookFile)
	if err != nil {
		guestbook = []GuestbookEntry{}
		return
	}
	json.Unmarshal(data, &guestbook)
}

// Saves entries to disk
func saveGuestbook() {
	data, _ := json.MarshalIndent(guestbook, "", "  ")

	err := os.WriteFile(guestbookFile, data, 0o644)
	util.HandleFSError(err, GuestbookDir, "saveGuestbook")
}

// Adds a new entry
func addGuestbookEntry(name string) GuestbookEntry {
	guestbookLock.Lock()
	defer guestbookLock.Unlock()

	entry := GuestbookEntry{
		Name:      name,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	guestbook = append(guestbook, entry)
	saveGuestbook()
	return entry
}

// Returns all entries
func getGuestbookEntries() []GuestbookEntry {
	guestbookLock.Lock()
	defer guestbookLock.Unlock()
	return guestbook
}

// ============ //
// API handling //
// ============ //

func GuestbookRouting(mux *http.ServeMux) {
	mux.HandleFunc("/api/guestbook", GuestbookHandler)
}

func GuestbookHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(getGuestbookEntries())

	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "invalid request"}`, http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, `{"error": "name required"}`, http.StatusBadRequest)
			return
		}

		entry := addGuestbookEntry(req.Name)
		w.WriteHeader(http.StatusCreated) // 201 Created is more accurate for POST
		json.NewEncoder(w).Encode(entry)

	default:
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
