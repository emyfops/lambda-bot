package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bwmarrin/discordgo"
	"github.com/zekrotja/ken"
	"io"
	"net/http"
	"slices"
	"strings"
)

type Capes struct{ ken.EphemeralCommand }

var _ ken.SlashCommand = (*Capes)(nil)
var _ ken.DmCapable = (*Capes)(nil)
var _ ken.GuildScopedCommand = (*Capes)(nil)

func (c *Capes) Name() string        { return "cape" }
func (c *Capes) Description() string { return "Manage Lambda capes" }
func (c *Capes) Version() string     { return "v1" }
func (c *Capes) IsDmCapable() bool   { return true }
func (c *Capes) Guild() string       { return *guildScope }

func (c *Capes) Run(ctx ken.Context) (err error) {
	err = ctx.HandleSubCommands(
		ken.SubCommandHandler{Name: "add", Run: addCape},
		ken.SubCommandHandler{Name: "remove", Run: removeCape})
	return
}

func addCape(ctx ken.SubCommandContext) (err error) {
	cape := ctx.Options().GetByName("cape").StringValue()
	fid := ctx.Options().GetByName("attachment").Value.(string)
	attachment := ctx.GetEvent().ApplicationCommandData().Resolved.Attachments[fid]

	resp, err := http.Get(attachment.URL)
	if err != nil {
		return err
	}

	var capes []string
	capes, err = getCapes()
	if err != nil {
		return
	}

	capes = append(capes, cape)
	err = updateCapes(capes)
	if err != nil {
		return
	}

	_, err = r2.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        bucketName,
		Key:           aws.String(cape + ".png"),
		Body:          resp.Body,
		ContentLength: aws.Int64(resp.ContentLength),
		ContentType:   aws.String("image/png"),
	})
	if err != nil {
		return
	}

	return ctx.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Added cape %s to the bucket", cape),
		},
	})
}

func removeCape(ctx ken.SubCommandContext) (err error) {
	cape := ctx.Options().GetByName("cape").StringValue()

	var capes []string
	capes, err = getCapes()
	if err != nil {
		return
	}

	slices.DeleteFunc(capes, func(s string) bool { return s == cape })
	err = updateCapes(capes)
	if err != nil {
		return
	}

	_, err = r2.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: bucketName,
		Key:    aws.String(cape + ".png"),
	})
	if err != nil {
		return
	}

	return ctx.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Removed cape " + cape,
		},
	})
}

func getCapes() ([]string, error) {
	result, err := r2.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: bucketName,
		Key:    aws.String("capes.txt"),
	})
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}

	return strings.Fields(string(bytes)), nil
}

func updateCapes(capes []string) (err error) {
	_, err = r2.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      bucketName,
		Key:         aws.String("capes.txt"),
		Body:        strings.NewReader(strings.TrimSpace(strings.Join(capes, "\n"))),
		ContentType: aws.String("plain/text"),
	})
	return
}

func (c *Capes) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "add",
			Description: "Uploads a cape to cloudflare",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "cape",
					Description: "Cape name without the extension",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionAttachment,
					Name:        "attachment",
					Description: "Cape image",
					Required:    true,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "remove",
			Description: "Removes a cape from cloudflare",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "cape",
					Description: "Cloudflare cape id",
					Required:    true,
				},
			},
		},
	}
}
