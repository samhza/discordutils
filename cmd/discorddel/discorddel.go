package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"samhza.com/discordutils/internal/archive"
	"samhza.com/discordutils/internal/token"
)

const (
	UnknownMessage                 httputil.ErrorCode = 10008
	SystemMessageActionUnavailable httputil.ErrorCode = 50021
	InvalidActionOnArchivedThread  httputil.ErrorCode = 50083
)

func main() {
	tok := flag.String("tok", "", "Discord user token")
	chid := flag.Uint64("channel", 0, "Discord channel ID")
	gid := flag.Uint64("guild", 0, "Discord guild ID")
	archiveDir := flag.String("archive", "", "directory to log deleted messages in")
	flag.Parse()
	if *chid == 0 && *gid == 0 {
		flag.Usage()
		log.Fatalln("at least one of -channel and -guild must be specified")
	}
	err := token.Get(tok)
	if err != nil {
		log.Fatalln(err)
	}
	var output *archive.Output
	if *archiveDir != "" {
		var err error
		output, err = archive.NewOutput(*archiveDir)
		if err != nil {
			log.Fatalln("Error while opening archive directory:", err)
		}
		defer output.Close()
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()
	c := session.New(*tok)
	self, err := c.Me()
	if err != nil {
		log.Fatalln("Error fetching self:", err)
	}
	pause := make(chan struct{})
	c.AddHandler(func(m *gateway.MessageCreateEvent) {
		if m.Author.ID == self.ID {
			pause <- struct{}{}
		}
	})
	if err := c.Open(ctx); err != nil {
		log.Fatalln(err)
	}
	defer c.Close()
	c = c.WithContext(ctx)
	searchdata := api.SearchData{
		SortBy:    "timestamp",
		SortOrder: "asc",
		AuthorID:  self.ID,
	}
	var guildID discord.GuildID
	if *chid != 0 {
		chid := discord.ChannelID(*chid)
		searchdata.ChannelID = chid
		ch, err := c.Channel(chid)
		if err != nil {
			log.Fatalln("Error while fetching channel: ", err)
		}
		guildID = ch.GuildID
	} else {
		guildID = discord.GuildID(*gid)
	}
	now := time.Now()
	var deleted uint = 0
Outer:
	for {
		var results api.SearchResponse
		if guildID.IsValid() {
			results, err = c.Client.Search(guildID, searchdata)
		} else {
			results, err = c.Client.SearchDirectMessages(searchdata)
		}
		if err != nil {
			log.Fatalln("Error occured while searching messages:", err)
		}
		log.Printf("%d messages remaining.\n", results.TotalResults)
		if deleted > 0 {
			log.Printf("Estimated remaining time: %s\n", time.Since(now)/time.Duration(deleted)*time.Duration(results.TotalResults))
		}
		if results.TotalResults == 0 {
			break Outer
		}
		for _, result := range results.Messages {
			for _, m := range result {
			Inner:
				select {
				case <-pause:
					timer := time.NewTimer(30 * time.Second)
					for {
						select {
						case <-timer.C:
							break Inner
						case <-pause:
							timer.Reset(30 * time.Second)
						case <-ctx.Done():
							break Outer
						}
					}
				case <-ctx.Done():
					break Outer
				default:
				}
				m.GuildID = discord.GuildID(*gid)
				if output != nil {
					err := output.LogMessage(m)
					if err != nil {
						log.Fatalf("Error logging message %s: %s\n", m.URL(), err)
					}
				}
				if m.Author.ID != self.ID {
					goto Continue
				}
				err = deleteMsg(c.Client, m)
				if err != nil {
					log.Printf("Error deleting %s: %s\n", m.URL(), err)
				}
			Continue:
				deleted++
				searchdata.MinID = m.ID + 1
			}
		}
	}
}

func deleteMsg(c *api.Client, m discord.Message) error {
	err := c.DeleteMessage(m.ChannelID, m.ID, "")
	if err == nil {
		return nil
	}
	var derr *httputil.HTTPError
	if ok := errors.As(err, &derr); ok {
		switch derr.Code {
		case UnknownMessage:
			return nil
		case SystemMessageActionUnavailable:
			return nil
		case InvalidActionOnArchivedThread:
			msg, err := c.SendMessage(m.ChannelID, "\u200B")
			if err != nil {
				return fmt.Errorf("sending message to unarchive thread %s: %w", chanURL(m.GuildID, m.ChannelID), err)
			}
			err = c.DeleteMessage(m.ChannelID, m.ID, "")
			if err != nil {
				return fmt.Errorf("deleting unarchive-trigger message %s: %w", msg.URL(), err)
			}
			return deleteMsg(c, m)
		}
	}
	return err
}

func chanURL(gid discord.GuildID, cid discord.ChannelID) string {
	var g string
	if gid.IsNull() {
		g = "@me"
	} else {
		g = gid.String()
	}
	return fmt.Sprintf("https://discord.com/channels/%s/%s", g, cid)
}
