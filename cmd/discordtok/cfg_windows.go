package main

import (
	"errors"
	"os"
)

func confighome() (cfghome string, err error) {
	var ok bool
	cfghome, ok = os.LookupEnv("APPDATA")
	if !ok {
		err = errors.New("%APPDATA% is not set")
	}
	return
}
