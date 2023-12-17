package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

//goland:noinspection GoUnusedConst
const (
	Windows = "windows"
	Mac     = "darwin"
	Linux   = "linux"
)

func GetCurrentDir() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}
	return filepath.Dir(ex), nil
}

//goland:noinspection GoUnusedExportedFunction
func OpenUrl(url string) error {
	var err error

	switch runtime.GOOS {
	case Linux:
		err = exec.Command("xdg-open", url).Start()
	case Windows:
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case Mac:
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		return fmt.Errorf("failed to open browser: %v", err)
	}

	return nil
}
