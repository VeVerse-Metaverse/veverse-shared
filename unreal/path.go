package unreal

import (
	"dev.hackerman.me/artheon/veverse-shared/config"
	"dev.hackerman.me/artheon/veverse-shared/platform"
	"fmt"
	"github.com/sirupsen/logrus"
	"os/user"
	"path/filepath"
	"runtime"
)

//goland:noinspection GoBoolExpressions,GoUnusedExportedFunction
func GetProjectSaveDir(project string, configuration string) (string, error) {
	if configuration == config.Shipping {
		usr, err := user.Current()
		if err != nil {
			logrus.Fatalf("failed to get current user: %v", err)
		}

		if runtime.GOOS == platform.Windows {
			return filepath.Join(usr.HomeDir, "AppData", "Local", project, "Saved"), nil
		} else if runtime.GOOS == platform.Linux {
			return filepath.Join(usr.HomeDir, ".config", project, "Saved"), nil
		} else if runtime.GOOS == platform.Mac {
			return filepath.Join(usr.HomeDir, "Library", "Application Support", project, "Saved"), nil
		}
	}

	currentDir, err := platform.GetCurrentDir()
	if err != nil {
		return "", fmt.Errorf("failed to get current dir: %v", err)
	}

	return filepath.Join(currentDir, project, project, "Saved"), nil
}
