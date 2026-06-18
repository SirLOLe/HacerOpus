package converter

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"os"
)

// FindBestCover busca la mejor imagen en la carpeta del archivo caótico
func FindBestCover(chaoticFolder string) string {
	covers := []string{"folder.jpg", "cover.jpg", "front.png", "cover.png"}
	for _, c := range covers {
		path := filepath.Join(chaoticFolder, c)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "" // No se encontró portada externa
}

// ProcessToOpus convierte el archivo aplicando todas las reglas estrictas
func ProcessToOpus(input string, output string, coverPath string, hasInternalCover bool) error {
	var args []string

	if coverPath != "" && !hasInternalCover {
		// Tiene portada externa, incrustar y escalar a 700x700
		args = []string{
			"-y", "-i", input, "-i", coverPath,
			"-map", "0:a", "-map", "1:v",
			"-c:a", "libopus", "-b:a", "128k", "-vbr", "on",
			"-c:v", "mjpeg", "-vf", "scale=700:700", "-disposition:v:0", "attached_pic",
			output,
		}
	} else {
		// Ya tiene portada interna o no tiene nada. Solo normalizar audio.
		args = []string{
			"-y", "-i", input,
			"-map", "0", // Mapear todo (audio + metadata + posible portada interna)
			"-c:a", "libopus", "-b:a", "128k", "-vbr", "on",
			"-c:v", "copy", // Copiar portada interna si existe
			output,
		}
	}

	cmd := exec.Command("ffmpeg", args...)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error FFmpeg en %s: %v", filepath.Base(input), err)
	}
	return nil
}