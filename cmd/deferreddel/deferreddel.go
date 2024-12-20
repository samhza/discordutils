package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"samhza.com/discordutils/internal/token"
)

var (
	ses     *session.Session
	dur     time.Duration
	userID  discord.UserID
	verbose bool
)

var (
	guildIDs map[discord.GuildID]struct{}
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
		guildIDs = make(map[discord.GuildID]struct{}, len(split))
		for _, s := range split {
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				log.Fatalln(err)
			}
			guildIDs[discord.GuildID(n)] = struct{}{}
		}
	}

	err := token.Get(tok)
	if err != nil {
		log.Fatalln(err)
	}

	ses = session.New(*tok)
	ctx, cancelCtx := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelCtx()

	err = ses.Open(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer ses.Close()

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

	run(ctx, evs)
}

func run(ctx context.Context, evs <-chan interface{}) {
	cancels := make(map[discord.MessageID]context.CancelFunc)
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-evs:
			switch ev := ev.(type) {
			case *gateway.MessageCreateEvent:
				if ev.Author.ID != userID {
					continue
				}
				if len(guildIDs) != 0 {
					if _, ok := guildIDs[ev.GuildID]; !ok {
						continue
					}
				}
				var mctx context.Context
				mctx, cancels[ev.ID] = context.WithCancel(ctx)
				go deferredDelete(mctx, ev.Message)
			case *gateway.MessageDeleteEvent:
				if cancel, ok := cancels[ev.ID]; ok {
					cancel()
					delete(cancels, ev.ID)
				}
			}
		}
	}
}

func deferredDelete(ctx context.Context, msg discord.Message) {
	when := msg.Timestamp.Time().Add(dur)
	vlog(msg.URL(), "will be deleted at", when)
	timer := time.NewTimer(time.Until(when))
	defer timer.Stop()

	select {
	case <-timer.C:
		if err := ses.DeleteMessage(msg.ChannelID, msg.ID, ""); err != nil {
			log.Println(err)
		} else {
			vlog(msg.URL(), "successfully deleted")
		}
	case <-ctx.Done():
		vlog(msg.URL(), "won't be deleted, already deleted")
	}
}
