package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/session"
	"samhza.com/discordutils/internal/token"
)

var (
	ses     *session.Session
	dur     time.Duration
	userID  discord.UserID
	verbose bool
)

var (
	guildIDs []discord.GuildID
)

func vlog(v ...interface{}) {
	if verbose {
		log.Println(v...)
	}
}

func main() {
	flag.BoolVar(&verbose, "v", false, "log deleted messages")
	gids := flag.String("gids", "", "guild IDs, comma separated")
	flag.DurationVar(&dur, "dur", 48*time.Hour, "delay for deleting messages")
	tok := flag.String("tok", "", "token")
	flag.Parse()
	if *gids != "" {
		split := strings.Split(*gids, ",")
		guildIDs = make([]discord.GuildID, len(split))
		for i, s := range split {
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				log.Fatalln(err)
			}
			guildIDs[i] = discord.GuildID(n)
		}
	}
	err := token.Get(tok)
	if err != nil {
		log.Fatalln(err)
	}
	ses, err = session.New(*tok)
	if err != nil {
		log.Fatalln(err)
	}
	err = ses.Open()
	if err != nil {
		log.Fatalln(err)
	}
	defer ses.CloseGracefully()
	me, err := ses.Me()
	if err != nil {
		log.Fatalln(err)
	}
	userID = me.ID
	evs, cancel := ses.ChanFor(
		func(ev interface{}) bool {
			switch ev.(type) {
			case *gateway.MessageCreateEvent:
				return true
			case *gateway.MessageDeleteEvent:
				return true
			default:
				return false
			}
		})
	defer cancel()
	go run(evs)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}

func run(evs <-chan interface{}) {
	cancels := make(map[discord.MessageID]context.CancelFunc)
Outer:
	for ev := range evs {
		switch ev := ev.(type) {
		case *gateway.MessageCreateEvent:
			var ctx context.Context
			if ev.Author.ID != userID {
				continue
			}
			if len(guildIDs) != 0 {
				for _, gid := range guildIDs {
					if ev.GuildID != gid {
						continue Outer
					}
				}
			}
			ctx, cancels[ev.ID] = context.WithCancel(context.Background())
			go deferredDelete(ev.Message, ctx)
		case *gateway.MessageDeleteEvent:
			if cancel, ok := cancels[ev.ID]; ok {
				cancel()
			}
		}
	}
}

func deferredDelete(msg discord.Message, ctx context.Context) {
	when := msg.Timestamp.Time().Add(dur)
	vlog(msg.URL(), "will be deleted at", when)
	timer := time.NewTimer(time.Until(when))
	defer timer.Stop()
	select {
	case <-timer.C:
		if err := ses.DeleteMessage(msg.ChannelID, msg.ID); err != nil {
			log.Println(err)
		} else {
			vlog(msg.URL(), "sucessfully deleted")
		}
	case <-ctx.Done():
		vlog(msg.URL(), "won't be deleted, already deleted")
	}
}
