package main

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/term"
	"echosystem/api"
	"echosystem/util"
	"echosystem/views"
)

//go:embed static/*
var staticFS embed.FS

func main() {
	log.Printf("=== echosystem site+server ver. %v ==", util.AppVersion)
	log.Println("Running initial verifications..")

	// = check for directories. ask user to create if needed. = //
	_, err := os.Stat(api.GuestbookDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			mkdirNeeded(api.GuestbookDir)
		} else {
			fmt.Printf("[ERROR] operation failed: %v\n", err)
			os.Exit(1)
		}
	}
	log.Println("Everything is alright")

	// ====== //
	// server //
	// ====== //
	api.Backup()
	mux := http.NewServeMux()

	// serving and handling filesystems //
	staticFiles, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("Failed to create static sub-filesystem: %v", err)
	}

	fileServer := http.FileServer(http.FS(staticFiles))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))
	// === //

	mux.HandleFunc("/{$}", desktopHandler)

	mux.HandleFunc("/", notFound)
	api.GuestbookRouting(mux)
	api.BackupTimer(36 * time.Hour)

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func desktopHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := views.DesktopRoot().Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func notFound(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := views.NotFound().Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ======= //
// helpers //
// ======= //
func cmdMkdir(sudo string, dir string) error {
	cmdMkdir := exec.Command("sudo", "-S", "mkdir", "-p", dir)
	userEnv := os.Getenv("USER")
	cmdChown := exec.Command("sudo", "-S", "chown", "-R", userEnv, dir)

	var stMkdir bytes.Buffer
	stMkdir.WriteString(sudo + "\n")
	cmdMkdir.Stdin = &stMkdir
	if err := cmdMkdir.Run(); err != nil {
		fmt.Printf("[FATAL] mkdir failed, password you entered is probably incorrect. Error code: %v \n", err)
		os.Exit(1)
	}

	var stChown bytes.Buffer
	stChown.WriteString(sudo + "\n")
	cmdChown.Stdin = &stChown
	if err := cmdChown.Run(); err != nil {
		fmt.Printf("[FATAL] chown failed: %v \n", err)
		os.Exit(1)
	}
	return nil
}

func mkdirNeeded(dir string) {
	fmt.Printf("[WARN] needed directory '%s' not found.\n", dir)
	fmt.Print("create directory now? (y/n + enter): ")
	scanner := bufio.NewScanner(os.Stdin)
	var cmd string
	if scanner.Scan() {
		cmd = strings.TrimSpace(strings.ToLower(scanner.Text()))
	}
	if cmd == "y" || cmd == "yes" {
		fmt.Printf("sudo password is needed for operation - create %s \n", dir)
		fmt.Print("type your sudo password (hidden): ")
		psswdBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Printf("\n[FATAL] couldn't read password safely: %v\n", err)
			os.Exit(1)
		}
		sudo := strings.TrimSpace(string(psswdBytes))
		fmt.Println()
		if errAuto := cmdMkdir(sudo, dir); errAuto != nil {
			fmt.Printf("\n[FATAL] couldn't make directory: %v\n", errAuto)
			fmt.Println("please, make the directory manually:")
			fmt.Printf("sudo mkdir -p %s && sudo chown -R $(USER) %s\n", dir, dir)
			os.Exit(1)
		}
		fmt.Println("[LOG] needed directories were made")
	} else {
		fmt.Println("[WARN] needed operation refused, exiting")
		os.Exit(1)
	}
}
