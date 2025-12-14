package main

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/agnivade/levenshtein"
	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger/v4"
	"github.com/pkg/errors"
	"github.com/zekrotja/ken"
)

type Config struct{ ken.EphemeralCommand }

var _ ken.SlashCommand = (*Config)(nil)
var _ ken.DmCapable = (*Config)(nil)
var _ ken.GuildScopedCommand = (*Config)(nil)

func (c *Config) Name() string        { return "snippet" }
func (c *Config) Description() string { return "Lambda snippets" }
func (c *Config) Version() string     { return "v1" }
func (c *Config) IsDmCapable() bool   { return true }
func (c *Config) Guild() string       { return *guildScope }

func (c *Config) Run(ctx ken.Context) error {
	return ctx.HandleSubCommands(
		ken.SubCommandHandler{Name: "create", Run: c.createSnippet},
		ken.SubCommandHandler{Name: "query", Run: c.querySnippet},
		ken.SubCommandHandler{Name: "list", Run: c.listSnippets})
}

func (c *Config) createSnippet(ctx ken.SubCommandContext) error {
	if !slices.Contains(*allowedUsers, ctx.User().ID) {
		return fmt.Errorf("you are not allowed to use this commannd")
	}

	name := strings.ToLower(ctx.Options().GetByName("name").StringValue())
	fileOption, ok1 := ctx.Options().GetByNameOptional("file")
	snippetOption, ok2 := ctx.Options().GetByNameOptional("snippet")
	overrideOpt, ok3 := ctx.Options().GetByNameOptional("override")
	override := ok3 && overrideOpt.BoolValue()

	if ok1 == false && ok2 == false {
		return errors.New("neither the 'file' or 'snippet' option was provided")
	}

	var snippet []byte

	if ok1 {
		fid := fileOption.Value.(string)
		attach := ctx.GetEvent().ApplicationCommandData().Resolved.Attachments[fid]
		resp, err := http.Get(attach.URL)
		if err != nil {
			return err
		}

		snippet, _ = io.ReadAll(resp.Body)
	} else if ok2 {
		snippet = []byte(snippetOption.StringValue())
	}

	return db.Update(func(txn *badger.Txn) error {
		_, e := txn.Get([]byte(name))
		if !errors.Is(e, badger.ErrKeyNotFound) && !override {
			return fmt.Errorf("there is already a snippet called '%s'. set the override option to true to override it", name)
		}

		txn.Set([]byte(name), snippet)

		return ctx.Respond(&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Added the config snippet '%s'", name),
			},
		})
	})
}

func (c *Config) querySnippet(ctx ken.SubCommandContext) error {
	name := strings.ToLower(ctx.Options().GetByName("name").StringValue())

	return db.View(func(txn *badger.Txn) error {
		itr := txn.NewIterator(badger.DefaultIteratorOptions)
		defer itr.Close()
		for itr.Rewind(); itr.Valid(); itr.Next() {
			item := itr.Item()

			key := strings.ToLower(string(item.Key()))

			if levenshtein.ComputeDistance(name, key) < 2 {
				i, _ := txn.Get([]byte(key))
				snippet, _ := i.ValueCopy(nil)

				ctx.SetEphemeral(false)
				return ctx.RespondEmbed(&discordgo.MessageEmbed{
					Description: fmt.Sprintf("```%s```", string(snippet)),
				})
			}
		}

		return errors.New(fmt.Sprintf("The snippet '%s' was not found.", name))
	})
}

func (c *Config) listSnippets(ctx ken.SubCommandContext) error {
	return db.View(func(txn *badger.Txn) error {
		snippets := make([]string, 0)
		itr := txn.NewIterator(badger.DefaultIteratorOptions)
		defer itr.Close()
		for itr.Rewind(); itr.Valid(); itr.Next() {
			snippets = append(snippets, string(itr.Item().Key()))
		}

		return ctx.RespondEmbed(&discordgo.MessageEmbed{
			Title:       "Available snippets",
			Description: "```" + strings.Join(snippets, ", ") + "```",
		})
	})
}

func (c *Config) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "create",
			Description: "Create a new config entry",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "The name of the snippet config",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionAttachment,
					Name:        "file",
					Description: "The snippet file",
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "snippet",
					Description: "The snippet content",
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "override",
					Description: "Override the snippet if it exists",
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "query",
			Description: "Query and return a config snippet",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "The name of the snippet",
					Required:    true,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "list",
			Description: "Return the list of available snippets",
		},
	}
}
