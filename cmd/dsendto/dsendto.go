package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/sendpart"
	"go.samhza.com/discordutils/internal/token"
)

func main() {
	var ch discord.ChannelID
	flag.Func("ch", "channel ID", func(str string) error {
		n, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return err
		}
		ch = discord.ChannelID(n)
		if !ch.IsValid() {
			return errors.New("channel is not valid")
		}
		return nil
	})
	n := flag.String("n", "stdout.txt", "file name")
	tok := flag.String("tok", "", "token")
	fname := flag.String("f", "-", "input file")
	flag.Parse()
	if ch == 0 {
		log.Fatalln("channel is not specified")
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
	m, err := c.SendMessageComplex(discord.ChannelID(ch), api.SendMessageData{
		Files: []sendpart.File{{Name: *n, Reader: f}},
	})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(m.URL())
}
