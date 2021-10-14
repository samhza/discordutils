package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/sendpart"
	"go.samhza.com/discordutils/internal/token"
)

func main() {
	ch := flag.Uint64("ch", 0, "channel ID")
	n := flag.String("n", "stdout.txt", "file name")
	tok := flag.String("tok", "", "token")
	fname := flag.String("f", "-", "input file")
	flag.Parse()
	if !discord.ChannelID(*ch).IsValid() {
		log.Fatalln("channel is not valid")
	}
	var f *os.File
	if *fname == "-" {
		f = os.Stdin
	} else {
		var err error
		f, err = os.Open(*fname)
		if err != nil {
			log.Fatalln(err)
		}
		defer f.Close()
	}
	if err := token.Get(tok); err != nil {
		log.Fatalln(err)
	}
	c := api.NewClient(*tok)
	m, err := c.SendMessageComplex(discord.ChannelID(*ch), api.SendMessageData{
		Files: []sendpart.File{{Name: *n, Reader: f}},
	})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(m.URL())
}
