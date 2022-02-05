//go:build !windows && !darwin
// +build !windows,!darwin

package main

import (
	"os"
	"os/user"
	"path/filepath"
)

func confighome() (string, error) {
	if cfghome, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		return cfghome, nil
	}
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(user.HomeDir, ".config"), nil
}
