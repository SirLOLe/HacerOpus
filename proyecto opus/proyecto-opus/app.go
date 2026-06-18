package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"music-normalizer/backend/converter"
	"music-normalizer/backend/metadata"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func NewApp() *App { return &App{} }
func (a *App) startup(ctx context.Context) { a.ctx = ctx }

func (a *App) StartSmartNormalization(inputRoot string, outputRoot string) string {
	var audioFiles []string

	runtime.EventsEmit(a.ctx, "status", "Analizando caos...")
	filepath.Walk(inputRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".mp3" || ext == ".flac" || ext == ".m4a" || ext == ".wav" || ext == ".opus" || ext == ".ogg" {
				audioFiles = append(audioFiles, path)
			}
		}
		return nil
	})

	total := len(audioFiles)
	if total == 0 { return "No se encontró música." }

	errores := 0
	for i, file := range audioFiles {
		runtime.EventsEmit(a.ctx, "progress", map[string]interface{}{
			"current": i + 1,
			"total":   total,
			"file":    filepath.Base(file),
			"status":  "Extrayendo Meta & Convirtiendo...",
		})

		meta := metadata.ExtractSmartMetadata(file)
		outPath := metadata.GenerateOutputPath(outputRoot, meta)
		cover := converter.FindBestCover(filepath.Dir(file))

		err := converter.ProcessToOpus(file, outPath, cover, meta.HasCover)
		if err != nil {
			errores++
			fmt.Printf("⚠️ ERROR LOG: %v\n", err)
		}
	}

	runtime.EventsEmit(a.ctx, "status", "¡Organización Completada!")
	return fmt.Sprintf("Finalizado: %d procesados, %d errores ignorados.", total, errores)
}