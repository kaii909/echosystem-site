package util

import (
	"errors"
	"log"
	"os"
)

// HandleFSError handle basic filesystem errors
func HandleFSError(err error, dirPath, actionContext string) {
	if err == nil {
		log.Printf("[LOG] '%v' completed succesfully", actionContext)
		return
	}

	switch {
	case errors.Is(err, os.ErrNotExist):
		log.Printf("[ERROR] needed directory doesnt exist: %s", dirPath)
	case errors.Is(err, os.ErrPermission):
		log.Printf("[ERROR] no write permissions for directory: %s", dirPath)
	default:
		log.Printf("[ERROR] internal I/O error: %v", err)
	}
	log.Printf("[WARN] '%s' action failed", actionContext)
}
