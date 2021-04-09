package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

func init() {
	log.SetFlags(0)
}

var versions = []string{"discord", "discordcanary", "discordptb"}

func main() {
	confighome, err := confighome()
	if err != nil {
		log.Fatalln(err)
	}
	for _, ver := range versions {
		path := filepath.Join(confighome,
			ver, "Local Storage/leveldb")
		tok := tokenFromLevelDB(path)
		if tok != "" {
			fmt.Println(tok)
			os.Exit(0)
		}
	}
	log.Fatalln("couldn't find token")
}

var tokenKeys = []string{
	"_https://discord.com\x00\x01token",
	"_https://discordapp.com\x00\x01token",
	"_https://ptb.discord.com\x00\x01token",
	"_https://ptb.discordapp.com\x00\x01token",
	"_https://canary.discord.com\x00\x01token",
	"_https://canary.discordapp.com\x00\x01token",
}

func tokenFromLevelDB(path string) string {
	db, err := leveldb.OpenFile(path, &opt.Options{
		ReadOnly: true,
	})
	if err != nil {
		return ""
	}
	defer db.Close()
	for _, key := range tokenKeys {
		data, err := db.Get([]byte(key), nil)
		if len(data) == 0 || err != nil {
			continue
		}
		return strings.Trim(string(data[1:]), "\"")
	}
	return ""
}
