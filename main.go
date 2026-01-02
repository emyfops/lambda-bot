package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger/v4"
	flag "github.com/spf13/pflag"
	"github.com/zekrotja/ken"
	"go.uber.org/zap"
)

var (
	token        = flag.String("token", "", "Bot Token")
	dbPath       = flag.String("dbPath", "/tmp/badger", "The directory in which the database is created")
	guildScope   = flag.String("guild", "", "Guild ID command scope")
	thread       = flag.String("thread", "", "Thread channel ID to send the template at")
	allowedUsers = flag.StringSlice("users", []string{}, "User IDs to be allowed")
	db           *badger.DB
)

const ThreadMessage = `
A good bug report shouldn't leave others needing to chase you up for more information. Therefore, we ask you to investigate carefully, collect information and describe the issue in detail in your report. Please complete the following steps in advance to help us fix any potential bug as fast as possible.

1. Make sure that you are using the latest version.
2. Determine if your bug is really a bug and not an error on your side e.g. using incompatible environment components/versions.
3. See if other users have experienced (and potentially already solved) the same issue you are having.

Upload the stack trace at <https://mclo.gs/> and collect the follwing information:
- Oerating System
- CPU Architecture. (x86, ARM, AArch64)
- Version of the Java runtime.
- Minecraft version
- Lambda version

Once your issue has been resolved, we kindly ask you to close the thread.`

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var err error
	flag.Parse()

	db, err = badger.Open(badger.DefaultOptions(*dbPath))
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	bot, err := discordgo.New("Bot " + *token)
	if err != nil {
		log.Fatal("Error creating Discord session", zap.Error(err))
	}

	defer bot.Close()

	k, err := ken.New(bot)
	if err != nil {
		log.Fatal("Error creating Discord session", zap.Error(err))
	}

	defer k.Unregister()

	must(k.RegisterCommands(
		new(Config),
	))

	must(k.RegisterMiddlewares(
		new(Middleware),
	))

	bot.AddHandler(func(s *discordgo.Session, ready *discordgo.Ready) {
		log.Printf("Logged in as '%s' (%s) @ %+v", ready.User.String(), ready.User.ID, time.Now())
	})

	bot.AddHandler(func(s *discordgo.Session, t *discordgo.ThreadCreate) {
		if t.ParentID != *thread || !t.NewlyCreated {
			return
		}

		s.ChannelMessageSend(t.Channel.ID, ThreadMessage)
	})

	must(bot.Open())

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}
