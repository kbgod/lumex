// Command gen parses the Telegram Bot API HTML documentation and writes a Go
// client package (object types, enums, unions, request structs, typed methods,
// constants, and helpers) into the output directory.
//
// Usage:
//
//	go run ./cmd/gen                          # fetch live docs → ./telegram/*.go
//	go run ./cmd/gen -file api                # parse a local HTML copy
//	go run ./cmd/gen -dir tg -package tg      # customise output dir and package
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kbgod/lumex/v2/internal/gen"
)

const defaultURL = "https://core.telegram.org/bots/api"

func main() {
	var (
		url      = flag.String("url", defaultURL, "URL of the Bot API documentation to fetch")
		file     = flag.String("file", "", "local HTML file to parse instead of fetching -url")
		dir      = flag.String("dir", ".", "output directory for the generated files")
		pkg      = flag.String("package", "lumex", "package name of the generated files")
		enums    = flag.Bool("enums", true, "generate string enum types for descriptive fixed-value fields")
		requests = flag.Bool("requests", true, "generate request payload structs and methods")
	)
	flag.Parse()

	src, origin, err := loadSource(*file, *url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load documentation:", err)
		os.Exit(1)
	}

	files, stats, err := gen.Generate(src, gen.Config{
		Package:  *pkg,
		Origin:   origin,
		Enums:    *enums,
		Requests: *requests,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "create output dir:", err)
		os.Exit(1)
	}
	for name, code := range files {
		if err := os.WriteFile(filepath.Join(*dir, name), code, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write output:", err)
			os.Exit(1)
		}
	}

	fmt.Printf("generated %d object types, %d enums, %d unions, %d request structs, %d methods → %s/\n",
		stats.ObjectTypes, stats.Enums, stats.Unions, stats.Requests, stats.Methods, *dir)
}

// loadSource returns the HTML to parse and a human-readable origin for the file
// header (the URL either fetched or associated with the local file).
func loadSource(file, url string) (string, string, error) {
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return "", "", err
		}
		return string(b), url, nil
	}

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "telegram-types-generator/1.0 (+https://core.telegram.org/bots/api)")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	return string(b), url, nil
}
