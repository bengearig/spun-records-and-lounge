package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	EnvChromium  = "SPUN_CHROMIUM"
	envNoSandbox = "SPUN_CHROMIUM_NO_SANDBOX"
)

var candidates = []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"}

// Find resolves the Chromium executable: flag, then SPUN_CHROMIUM, then PATH.
func Find(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if env := os.Getenv(EnvChromium); env != "" {
		return env, nil
	}
	for _, name := range candidates {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("chromium not found; checked the --chromium flag, $%s, and PATH for %s. "+
		"Install Chromium (e.g. `sudo apt-get install chromium`) or set --chromium/%s to its path",
		EnvChromium, strings.Join(candidates, ", "), EnvChromium)
}
