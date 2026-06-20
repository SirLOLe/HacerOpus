package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

type ConversionSettings struct {
	Bitrate          string `json:"bitrate"`
	CoverSize        int    `json:"coverSize"`
	Quality          int    `json:"quality"`
	ReplaceCover     bool   `json:"replaceCover"`
	CompressionLevel int    `json:"compressionLevel"`
	Threads          int    `json:"threads"`
}

type ProgressUpdate struct {
	CurrentFile string  `json:"currentFile"`
	Progress    float64 `json:"progress"`
	Processed   int     `json:"processed"`
	Total       int     `json:"total"`
	Errors      int     `json:"errors"`
}

type TrackMetadata struct {
	Artist string
	Album  string
	Title  string
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) SelectDirectory(title string) string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		return ""
	}
	return dir
}

func getMetadata(filePath string) TrackMetadata {
	fileName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	dirName := filepath.Base(filepath.Dir(filePath))

	meta := TrackMetadata{
		Artist: "Desconocido",
		Album:  "Desconocido",
		Title:  fileName,
	}

	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", filePath)
	output, err := cmd.Output()
	if err == nil {
		var data struct {
			Format struct {
				Tags map[string]string `json:"tags"`
			} `json:"format"`
		}
		if json.Unmarshal(output, &data) == nil {
			for k, v := range data.Format.Tags {
				switch strings.ToLower(k) {
				case "artist", "album_artist", "albumartist":
					if meta.Artist == "Desconocido" {
						meta.Artist = v
					}
				case "album":
					meta.Album = v
				case "title":
					meta.Title = v
				}
			}
		}
	}

	// Fallback: parsear nombre de archivo si faltan tags
	if meta.Title == fileName || meta.Title == "" {
		if parts := strings.SplitN(fileName, " - ", 2); len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			isNum := true
			for _, c := range strings.TrimRight(left, ".") {
				if c < '0' || c > '9' {
					isNum = false
					break
				}
			}
			if isNum {
				meta.Title = right
			} else {
				if meta.Artist == "Desconocido" {
					meta.Artist = left
				}
				meta.Title = right
			}
		}
	}

	// Fallback: usar carpeta como álbum
	if meta.Album == "Desconocido" && dirName != "." && dirName != "" {
		meta.Album = dirName
	}

	// Limpiar caracteres inválidos en Windows
	r := strings.NewReplacer(
		"<", "", ">", "", ":", "", "\"", "",
		"/", "-", "\\", "-", "|", "", "?", "", "*", "",
	)
	meta.Artist = strings.TrimSpace(r.Replace(meta.Artist))
	meta.Album = strings.TrimSpace(r.Replace(meta.Album))
	meta.Title = strings.TrimSpace(r.Replace(meta.Title))

	if meta.Artist == "" { meta.Artist = "Desconocido" }
	if meta.Album  == "" { meta.Album  = "Desconocido" }
	if meta.Title  == "" { meta.Title  = fileName }

	return meta
}

func (a *App) ConvertLibrary(inputDir string, outputDir string, settings ConversionSettings) string {
	var audioFiles []string
	validExts := map[string]bool{
		".mp3": true, ".flac": true, ".m4a": true,
		".wav": true, ".ogg": true, ".opus": true,
	}

	filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if !info.IsDir() {
			if validExts[strings.ToLower(filepath.Ext(path))] {
				audioFiles = append(audioFiles, path)
			}
		}
		return nil
	})

	total := len(audioFiles)
	if total == 0 {
		return "No se encontraron archivos de audio."
	}

	threads := settings.Threads
	if threads < 1 { threads = 1 }

	sem := make(chan struct{}, threads)
	var wg sync.WaitGroup
	var processed, errorsCount int
	var mu sync.Mutex

	for _, file := range audioFiles {
		wg.Add(1)
		sem <- struct{}{}

		go func(filePath string) {
			defer wg.Done()
			defer func() { <-sem }()

			meta := getMetadata(filePath)

			outDirPath := filepath.Join(outputDir, meta.Artist, meta.Album)
			os.MkdirAll(outDirPath, os.ModePerm)
			outPath := filepath.Join(outDirPath, meta.Title+".opus")

			// Saltar si ya existe
			if _, err := os.Stat(outPath); err == nil {
				mu.Lock()
				processed++
				progress := (float64(processed) / float64(total)) * 100
				runtime.EventsEmit(a.ctx, "conversion_progress", ProgressUpdate{
					CurrentFile: filepath.Base(filePath) + " (omitido)",
					Progress:    progress, Processed: processed,
					Total: total, Errors: errorsCount,
				})
				mu.Unlock()
				return
			}

			// Comando FFmpeg limpio — solo audio, sin portada
			// (Opus/Ogg no soporta streams de video incrustados)
			args := []string{
				"-y",
				"-i", filePath,
				"-vn",   // Ignorar cualquier stream de video/portada del input
				"-c:a", "libopus",
				"-b:a", settings.Bitrate,
				"-vbr", "on",
				"-compression_level", fmt.Sprintf("%d", settings.CompressionLevel),
				"-ar", "48000",
				"-threads", "1",
				"-map_metadata", "0",
				outPath,
			}

			cmd := exec.Command("ffmpeg", args...)
			var stderr strings.Builder
			cmd.Stderr = &stderr

			err := cmd.Run()

			mu.Lock()
			processed++
			if err != nil {
				errorsCount++
				// Log del error real para debuggear si vuelve a fallar
				fmt.Printf("ERROR [%s]: %v\n→ %s\n",
					filepath.Base(filePath), err, stderr.String())
			}
			progress := (float64(processed) / float64(total)) * 100
			runtime.EventsEmit(a.ctx, "conversion_progress", ProgressUpdate{
				CurrentFile: filepath.Base(filePath),
				Progress:    progress, Processed: processed,
				Total: total, Errors: errorsCount,
			})
			mu.Unlock()
		}(file)
	}

	wg.Wait()
	return fmt.Sprintf("Conversión finalizada. %d procesados, %d errores.", total, errorsCount)
}