package api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"echosystem/util"
)

// guestbookBackup creates a timestamped backup of the current in-memory guestbook data.
func guestbookBackup() {
	backupDir := filepath.Join(GuestbookDir, ".backup")
	
	// create backup directory if it doesn't exist
	if errMkdir := os.MkdirAll(backupDir, 0o755); errMkdir != nil {
		util.HandleFSError(errMkdir, backupDir, "backupDir creation")
		return 
	}

	guestbookLock.Lock()
	
	// Marshal directly from memory (much faster/safer than reading disk under lock)
	data, err := json.MarshalIndent(guestbook, "", "  ")
	
	guestbookLock.Unlock()
	
	if err != nil {
		log.Printf("[ERROR] failed to marshal guestbook for backup: %v", err)
		return
	}
	
	if len(data) == 0 {
		return
	}

	// Perform file I/O OUTSIDE the lock
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	fileName := fmt.Sprintf("guestbook_%s.json.bak", timestamp)
	dailyBackupFile := filepath.Join(backupDir, fileName)
	
	if errBak := os.WriteFile(dailyBackupFile, data, 0o644); errBak != nil {
		util.HandleFSError(errBak, backupDir, "guestbook dailyBackupFile")
	}
}

func Backup() {
	guestbookBackup()
	log.Println("[LOG] a backup has been made")
}

// BackupTimer starts a goroutine that performs backups at the specified interval.
func BackupTimer(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			Backup()
		}
	}()
}
