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

// Actualizamos el struct para recibir los nuevos parámetros de Svelte
type ConversionSettings struct {
	Bitrate          string `json:"bitrate"`
	CoverSize        int    `json:"coverSize"`
	Quality          int    `json:"quality"`
	ReplaceCover     bool   `json:"replaceCover"`
	CompressionLevel int    `json:"compressionLevel"` // 0 a 10
	Threads          int    `json:"threads"`          // Cantidad dinámica
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

func NewApp() *App {
	return &App{}
}

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
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", filePath)
	output, err := cmd.Output()
	
	meta := TrackMetadata{Artist: "Desconocido", Album: "Desconocido", Title: filepath.Base(filePath)}
	if err != nil {
		return meta
	}

	var data struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &data); err == nil {
		for k, v := range data.Format.Tags {
			lowerK := strings.ToLower(k)
			if lowerK == "artist" {
				meta.Artist = v
			} else if lowerK == "album" {
				meta.Album = v
			} else if lowerK == "title" {
				meta.Title = v
			}
		}
	}
	
	replacer := strings.NewReplacer("<", "", ">", "", ":", "", "\"", "", "/", "-", "\\", "-", "|", "", "?", "", "*", "")
	meta.Artist = strings.TrimSpace(replacer.Replace(meta.Artist))
	meta.Album = strings.TrimSpace(replacer.Replace(meta.Album))
	meta.Title = strings.TrimSpace(replacer.Replace(meta.Title))
	
	return meta
}

func findLocalCover(dir string) string {
	coverNames := []string{"cover.jpg", "cover.png", "folder.jpg", "folder.png", "front.jpg", "front.png"}
	for _, name := range coverNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func (a *App) ConvertLibrary(inputDir string, outputDir string, settings ConversionSettings) string {
	var audioFiles []string
	validExts := map[string]bool{".mp3": true, ".flac": true, ".m4a": true, ".wav": true, ".ogg": true, ".opus": true}

	filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if validExts[ext] {
				audioFiles = append(audioFiles, path)
			}
		}
		return nil
	})

	total := len(audioFiles)
	if total == 0 {
		return "No se encontraron archivos de audio."
	}

	var wg sync.WaitGroup
	
	// PROTECCIÓN: Si el usuario manda 0 hilos, forzamos a 1
	threads := settings.Threads
	if threads < 1 {
		threads = 1
	}
	// Aplicamos el límite de hilos dinámico
	sem := make(chan struct{}, threads)
	
	var processed int
	var errorsCount int
	var mu sync.Mutex

	for _, file := range audioFiles {
		wg.Add(1)
		sem <- struct{}{}

		go func(filePath string) {
			defer wg.Done()

			meta := getMetadata(filePath)
			
			outDirPath := filepath.Join(outputDir, meta.Artist, meta.Album)
			os.MkdirAll(outDirPath, os.ModePerm)
			outName := meta.Title + ".opus"
			outPath := filepath.Join(outDirPath, outName)

			if _, err := os.Stat(outPath); err == nil {
    mu.Lock()
    processed++
    progress := (float64(processed) / float64(total)) * 100
    runtime.EventsEmit(a.ctx, "conversion_progress", ProgressUpdate{
        CurrentFile: filepath.Base(filePath) + " (omitido)",
        Progress:    progress,
        Processed:   processed,
        Total:       total,
        Errors:      errorsCount,
    })
    mu.Unlock()
    <-sem
    return
}

			coverToUse := ""
			tempDir := filepath.Join(os.TempDir(), "haceropus_temp")
			os.MkdirAll(tempDir, os.ModePerm)
			extractedCover := filepath.Join(tempDir, filepath.Base(filePath)+"_cover.jpg")

			extractCmd := exec.Command("ffmpeg", "-y", "-i", filePath, "-an", "-vcodec", "copy", extractedCover)
			if err := extractCmd.Run(); err == nil {
				coverToUse = extractedCover
			} else {
				coverToUse = findLocalCover(filepath.Dir(filePath))
			}

			ffmpegArgs := []string{"-y", "-i", filePath}

			if coverToUse != "" {
				resizedCover := filepath.Join(tempDir, filepath.Base(filePath)+"_resized.jpg")
				scaleFilter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", 
					settings.CoverSize, settings.CoverSize, settings.CoverSize, settings.CoverSize)
				
				resizeCmd := exec.Command("ffmpeg", "-y", "-i", coverToUse, "-vf", scaleFilter, "-q:v", fmt.Sprintf("%d", settings.Quality), resizedCover)
				resizeCmd.Run()

				ffmpegArgs = append(ffmpegArgs, "-i", resizedCover, "-map", "0:a", "-map", "1:v")
			} else {
				ffmpegArgs = append(ffmpegArgs, "-map", "0:a")
			}

			// MEJORA: Se añade -compression_level y -ar 48000 para optimización nativa Opus
			ffmpegArgs = append(ffmpegArgs, 
				"-threads", "1",
				"-c:a", "libopus", 
				"-b:a", settings.Bitrate, 
				"-vbr", "on", 
				"-compression_level", fmt.Sprintf("%d", settings.CompressionLevel), 
				"-ar", "48000",
				"-map_metadata", "0")
			
			if coverToUse != "" {
				ffmpegArgs = append(ffmpegArgs, "-c:v", "copy", "-disposition:v", "attached_pic")
			}
			
			ffmpegArgs = append(ffmpegArgs, outPath)

			cmd := exec.Command("ffmpeg", ffmpegArgs...)
			err := cmd.Run()

			os.Remove(extractedCover)
			if coverToUse != "" {
				os.Remove(filepath.Join(tempDir, filepath.Base(filePath)+"_resized.jpg"))
			}

			mu.Lock()
			processed++
			if err != nil {
				errorsCount++
			}
			
			progress := (float64(processed) / float64(total)) * 100
			update := ProgressUpdate{
				CurrentFile: filepath.Base(filePath),
				Progress:    progress,
				Processed:   processed,
				Total:       total,
				Errors:      errorsCount,
			}
			
			runtime.EventsEmit(a.ctx, "conversion_progress", update)
			mu.Unlock()

			<-sem
		}(file)
	}

	wg.Wait()
	return fmt.Sprintf("Conversión masiva finalizada. %d procesados, %d errores.", total, errorsCount)
}