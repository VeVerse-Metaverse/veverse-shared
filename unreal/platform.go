package unreal

import (
	"dev.hackerman.me/artheon/veverse-shared/platform"
	"runtime"
)

const (
	Windows = "Win64"
	Mac     = "Mac"
	Linux   = "Linux"
)

//goland:noinspection GoUnusedExportedFunction, GoBoolExpressions
func GetPlatformName() string {
	if runtime.GOOS == platform.Windows {
		return Windows
	} else if runtime.GOOS == platform.Mac {
		return Mac
	} else if runtime.GOOS == platform.Linux {
		return Linux
	}
	return ""
}
