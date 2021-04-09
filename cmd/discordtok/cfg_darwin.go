package main

import (
	"os/user"
	"path/filepath"
)

func confighome() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(user.HomeDir,
		"Library", "Application Support"), nil
}
