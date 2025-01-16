package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"samhza.com/discordutils/internal/archive"
	"samhza.com/discordutils/internal/token"
)

var (
	ses     *session.Session
	dur     time.Duration
	userID  discord.UserID
	verbose bool
)

var (
	guildIDs = make(map[discord.GuildID]struct{})
)

func vlog(v ...interface{}) {
	if verbose {
		log.Println(v...)
	}
}

func main() {
	flag.BoolVar(&verbose, "v", false, "log queued message deletions")
	flag.Func("g", "guild ID to delete messages from (can be specified multiple times)", func(s string) error {
		n, err := discord.ParseSnowflake(s)
		guildIDs[discord.GuildID(n)] = struct{}{}
		return err
	})
	flag.DurationVar(&dur, "dur", 48*time.Hour, "delay for deleting messages")
	tok := flag.String("tok", "", "token")
	archiveDir := flag.String("archive", "", "directory to log deleted messages in")
	flag.Parse()

	if dur < 0 {
		log.Println("specified duration must be positive")
		flag.Usage()
		os.Exit(2)
	}

	err := token.Get(tok)
	if err != nil {
		log.Fatalln(err)
	}

	ses = session.New(*tok)
	ctx, cancelCtx := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelCtx()

	err = ses.Connect(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer ses.Close()
	ses = ses.WithContext(ctx)

	me, err := ses.Me()
	if err != nil {
		log.Fatalln(err)
	}
	userID = me.ID

	var output *archive.Output
	if *archiveDir != "" {
		output, err = archive.NewOutput(*archiveDir)
		if err != nil {
			log.Fatalln("Error while opening archive directory:", err)
		}
		defer output.Close()
	}

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

	run(ctx, evs, output)
}

func run(ctx context.Context, evs <-chan interface{}, output *archive.Output) {
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
				go deferredDelete(mctx, ev.Message, output)
			case *gateway.MessageDeleteEvent:
				if cancel, ok := cancels[ev.ID]; ok {
					cancel()
					delete(cancels, ev.ID)
				}
			}
		}
	}
}

func deferredDelete(ctx context.Context, msg discord.Message, output *archive.Output) {
	when := msg.Timestamp.Time().Add(dur)
	vlog(msg.URL(), "will be deleted at", when)
	timer := time.NewTimer(time.Until(when))
	defer timer.Stop()

	select {
	case <-timer.C:
		if output != nil {
			if err := output.LogMessage(msg); err != nil {
				log.Printf("Error logging message %s: %s\n", msg.URL(), err)
			}
		}
		if err := ses.DeleteMessage(msg.ChannelID, msg.ID, ""); err != nil {
			log.Println(err)
		} else {
			vlog(msg.URL(), "successfully deleted")
		}
	case <-ctx.Done():
		vlog(msg.URL(), "won't be deleted, already deleted")
	}
}
