package voice

import (
	"os"
	"os/exec"
	"path/filepath"
)

// PlayWAV reproduce un archivo WAV nativamente en Windows usando PowerShell
func PlayWAV(wavData []byte) error {
	tmpFile := filepath.Join(os.TempDir(), "ozy_speech_out.wav")
	if err := os.WriteFile(tmpFile, wavData, 0644); err != nil {
		return err
	}
	cmd := exec.Command("powershell", "-WindowStyle", "Hidden", "-Command", "(New-Object System.Media.SoundPlayer '"+tmpFile+"').PlaySync()")
	return cmd.Run()
}
