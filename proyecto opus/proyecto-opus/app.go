package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
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
	Artist   string
	Album    string
	Title    string
	HasCover bool
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
		Artist:   "Desconocido",
		Album:    "Desconocido",
		Title:    fileName,
		HasCover: false,
	}

	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", filePath)
	output, err := cmd.Output()
	if err == nil {
		var data struct {
			Streams []struct {
				CodecType string `json:"codec_type"`
			} `json:"streams"`
			Format struct {
				Tags map[string]string `json:"tags"`
			} `json:"format"`
		}
		if json.Unmarshal(output, &data) == nil {
			for _, stream := range data.Streams {
				if stream.CodecType == "video" {
					meta.HasCover = true
					break
				}
			}
			
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

	if meta.Album == "Desconocido" && dirName != "." && dirName != "" {
		meta.Album = dirName
	}

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

func findBestCover(folder string) string {
	covers := []string{"folder.jpg", "cover.jpg", "front.png", "cover.png", "Folder.jpg", "Cover.jpg", "Front.png", "Cover.png"}
	for _, c := range covers {
		path := filepath.Join(folder, c)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// createMetadataBlockPicture construye el bloque binario requerido por Ogg/Opus
func createMetadataBlockPicture(imageBytes []byte) string {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, uint32(3)) // 3 = Front Cover
	
	mimeType := "image/jpeg"
	binary.Write(buf, binary.BigEndian, uint32(len(mimeType)))
	buf.WriteString(mimeType)

	description := ""
	binary.Write(buf, binary.BigEndian, uint32(len(description)))
	buf.WriteString(description)

	// Anchura, altura, color, indexado (Al estar en 0, el reproductor las infiere automáticamente)
	binary.Write(buf, binary.BigEndian, uint32(0))
	binary.Write(buf, binary.BigEndian, uint32(0))
	binary.Write(buf, binary.BigEndian, uint32(0))
	binary.Write(buf, binary.BigEndian, uint32(0))

	binary.Write(buf, binary.BigEndian, uint32(len(imageBytes)))
	buf.Write(imageBytes)

	return base64.StdEncoding.EncodeToString(buf.Bytes())
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

			currentDir := filepath.Dir(filePath)
			externalCover := findBestCover(currentDir)

			coverSize := settings.CoverSize
			if coverSize <= 0 { coverSize = 700 }
			vfOpt := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", coverSize, coverSize)

			var imageBytes []byte

			// 1. Extraer la imagen en memoria (ya escalada)
			if (settings.ReplaceCover || !meta.HasCover) && externalCover != "" {
				extractCmd := exec.Command("ffmpeg", "-y", "-i", externalCover, "-vf", vfOpt, "-c:v", "mjpeg", "-f", "image2pipe", "-")
				imageBytes, _ = extractCmd.Output()
			} else if meta.HasCover {
				extractCmd := exec.Command("ffmpeg", "-y", "-i", filePath, "-an", "-vframes", "1", "-vf", vfOpt, "-c:v", "mjpeg", "-f", "image2pipe", "-")
				imageBytes, _ = extractCmd.Output()
			}

			args := []string{
				"-y",
				"-i", filePath,
			}

			// 2. Si conseguimos extraer una imagen, preparamos el archivo de metadatos especial para evitar desbordar los límites de Windows
			if len(imageBytes) > 0 {
				metaFile, err := os.CreateTemp("", "opusmeta_*.txt")
				if err == nil {
					metaFilePath := metaFile.Name()
					metaFile.Close()
					defer os.Remove(metaFilePath)

					// Volcar los metadatos originales al archivo temporal
					exec.Command("ffmpeg", "-y", "-i", filePath, "-f", "ffmetadata", metaFilePath).Run()

					metaData, _ := os.ReadFile(metaFilePath)
					metaDataStr := string(metaData)

					// Evitar posibles duplicados y estructurar
					lines := strings.Split(metaDataStr, "\n")
					var cleanLines []string
					for _, line := range lines {
						if !strings.HasPrefix(line, "METADATA_BLOCK_PICTURE=") && strings.TrimSpace(line) != "" {
							cleanLines = append(cleanLines, line)
						}
					}

					base64Str := createMetadataBlockPicture(imageBytes)
					base64Str = strings.ReplaceAll(base64Str, "=", "\\=") // Escapado obligatorio de FFmpeg

					cleanLines = append(cleanLines, "METADATA_BLOCK_PICTURE="+base64Str)
					os.WriteFile(metaFilePath, []byte(strings.Join(cleanLines, "\n")+"\n"), 0644)

					// Inyectar el archivo mapeado
					args = append(args, "-i", metaFilePath, "-map", "0:a", "-map_metadata", "1")
				} else {
					args = append(args, "-map", "0:a", "-map_metadata", "0")
				}
			} else {
				args = append(args, "-map", "0:a", "-map_metadata", "0")
			}

			// 3. Ejecutar la codificación final eliminando cualquier canal de video nativo para evitar crasheos de Opus
			args = append(args,
				"-c:a", "libopus",
				"-b:a", settings.Bitrate,
				"-vbr", "on",
				"-compression_level", fmt.Sprintf("%d", settings.CompressionLevel),
				"-ar", "48000",
				"-vn", // Remueve flujos de video. Es fundamental para Opus.
				"-threads", "1",
				outPath,
			)

			cmd := exec.Command("ffmpeg", args...)
			var stderr strings.Builder
			cmd.Stderr = &stderr

			err := cmd.Run()

			mu.Lock()
			processed++
			if err != nil {
				errorsCount++
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