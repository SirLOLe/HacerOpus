package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
)

type SongMeta struct {
	Artist      string
	Album       string
	Title       string
	Track       int
	Genre       string
	Year        int
	HasCover    bool
	OriginalExt string
}

// ExtractSmartMetadata analiza el caos y devuelve datos limpios
func ExtractSmartMetadata(filePath string) SongMeta {
	meta := SongMeta{OriginalExt: filepath.Ext(filePath)}
	fileName := strings.TrimSuffix(filepath.Base(filePath), meta.OriginalExt)
	dirName := filepath.Base(filepath.Dir(filePath))

	// 1. Intentar leer tags reales (ID3, Vorbis)
	file, err := os.Open(filePath)
	if err == nil {
		defer file.Close()
		m, err := tag.ReadFrom(file)
		if err == nil {
			meta.Title = m.Title()
			meta.Artist = m.Artist()
			meta.Album = m.Album()
			meta.Genre = m.Genre()
			meta.Track, _ = m.Track()
			meta.Year = m.Year()
			if m.Picture() != nil {
				meta.HasCover = true
			}
		}
	}

	// 2. Reglas de Smart Parsing (Fallback)
	if meta.Title == "" {
		// Inferir desde el nombre del archivo (ej: "01 - Mi Cancion" -> "Mi Cancion")
		parts := strings.SplitN(fileName, "-", 2)
		if len(parts) == 2 {
			meta.Title = strings.TrimSpace(parts[1])
		} else {
			meta.Title = fileName
		}
	}
	if meta.Artist == "" {
		meta.Artist = "Unknown Artist"
	}
	if meta.Album == "" {
		// Inferir desde el nombre de la carpeta
		if dirName != "." && dirName != "" {
			meta.Album = dirName
		} else {
			meta.Album = "Unknown Album"
		}
	}
	if meta.Track == 0 {
		meta.Track = 1 // Por defecto para evitar desorden
	}

	return meta
}

// GenerateOutputPath crea la estructura: /Salida/Artista/Album/01 - Titulo.opus
func GenerateOutputPath(outRoot string, meta SongMeta) string {
	artistFolder := filepath.Join(outRoot, meta.Artist)
	albumFolder := filepath.Join(artistFolder, meta.Album)
	os.MkdirAll(albumFolder, os.ModePerm)

	cleanTitle := strings.ReplaceAll(meta.Title, "/", "_") // Evitar errores de ruta
	fileName := fmt.Sprintf("%02d - %s.opus", meta.Track, cleanTitle)
	
	return filepath.Join(albumFolder, fileName)
}