package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bwmarrin/discordgo"
	flag "github.com/spf13/pflag"
	"github.com/zekrotja/ken"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	guildScope   = flag.String("guild", "", "Guild ID command scope")
	allowedUsers = flag.StringArray("users", []string{}, "User IDs to be allowed")
	token        = flag.String("token", "", "Bot Token")
	bucketUrl    = flag.String("bucketUrl", "", "Jurisdiction-specific endpoints for S3 clients")
	bucketKey    = flag.String("bucketKey", "", "")
	bucketSecret = flag.String("bucketSecret", "", "")
	bucketName   = flag.String("bucketName", "", "")
	bucketRegion = flag.String("bucketRegion", "ENAM", "")
	r2           *s3.Client
)

func main() {
	var err error
	flag.Parse()

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(*bucketRegion),
		config.WithBaseEndpoint(*bucketUrl),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*bucketKey, *bucketSecret, "")),
	)

	r2 = s3.NewFromConfig(cfg)

	bot, err := discordgo.New("Bot " + *token)
	if err != nil {
		log.Fatal("Error creating Discord session", zap.Error(err))
	}

	k, err := ken.New(bot)
	if err != nil {
		log.Fatal("Error creating Discord session", zap.Error(err))
	}

	must(k.RegisterCommands(
		new(Mappings),
		new(Capes),
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

	k.Unregister()
	bot.Close()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
