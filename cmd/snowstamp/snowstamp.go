package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/diamondburned/arikawa/v3/discord"
)

func main() {
	if len(os.Args) < 2 {
		log.Printf("%s <snowflake> [snowflake...]\n", os.Args[0])
		os.Exit(1)
	}
	for _, sf := range os.Args[1:] {
		n, err := strconv.ParseInt(sf, 10, 64)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		stamp := discord.Snowflake(n).Time()
		fmt.Println(stamp.Format("Mon, 02 Jan 2006 15:04:05.000"))
	}
}
