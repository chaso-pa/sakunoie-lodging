package config

import (
	"errors"
	"io/fs"
	"os"

	"github.com/joho/godotenv"
)

// LoadDotEnv copies variables from the dotenv file at path into the process
// environment for local development. Variables that are already set always
// win (godotenv.Load never overrides), so systemd EnvironmentFile, Docker env
// or an inline export take precedence. A missing file is not an error.
// SKIP_DOTENV=1 disables loading entirely (the E2E stack sets it so the
// shared-dev .env can never leak into a test run).
// It reports whether a file was loaded; values are never logged.
func LoadDotEnv(path string) (bool, error) {
	if os.Getenv("SKIP_DOTENV") == "1" {
		return false, nil
	}
	if err := godotenv.Load(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
