package token

import (
	"os"
	"path/filepath"
	"strings"
)

func Get(tok *string) error {
	if *tok == "" {
		cfg, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		tokb, err := os.ReadFile(filepath.Join(cfg, "discord-token"))
		if err != nil {
			return err
		}
		*tok = strings.TrimSpace(string(tokb))
	}
	return nil
}
