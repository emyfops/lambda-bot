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
	guildScope   = flag.String("guild", "", "Guild ID command scope")
	allowedUsers = flag.StringArray("users", []string{}, "User IDs to be allowed")
	token        = flag.String("token", "", "Bot Token")
	dbPath       = flag.String("dbPath", "/tmp/badger", "The directory in which the database is created")
	db           *badger.DB
)

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

	must(bot.Open())

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
