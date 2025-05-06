package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	tinyv2parser "github.com/Edouard127/tiny-v2-parser"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/zekrotja/ken"
	"io"
	"maps"
	"net/http"
	"strings"
)

type Mappings struct{ ken.EphemeralCommand }

var _ ken.SlashCommand = (*Mappings)(nil)
var _ ken.DmCapable = (*Mappings)(nil)
var _ ken.GuildScopedCommand = (*Mappings)(nil)

func (m *Mappings) Name() string        { return "mappings" }
func (m *Mappings) Description() string { return "Manage Lambda mappings" }
func (m *Mappings) Version() string     { return "v1" }
func (m *Mappings) IsDmCapable() bool   { return true }
func (m *Mappings) Guild() string       { return *guildScope }

func (m *Mappings) Run(ctx ken.Context) (err error) {
	err = ctx.HandleSubCommands(
		ken.SubCommandHandler{Name: "convert", Run: convertMappings},
		ken.SubCommandHandler{Name: "update", Run: updateMappings})
	return
}

func convertMappings(ctx ken.SubCommandContext) (err error) {
	version := ctx.Options().GetByName("version").StringValue()

	var data *bytes.Buffer
	data, err = downloadAndProcessMappings(version)
	if err != nil {
		return
	}

	return ctx.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Mappings processed successfully",
			Files: []*discordgo.File{
				{
					ContentType: "text/plain",
					Name:        version,
					Reader:      data,
				},
			},
		},
	})
}

func updateMappings(ctx ken.SubCommandContext) (err error) {
	for version := range maps.Keys(yarnMappings) {
		_, err = downloadAndProcessMappings(version)
		if err != nil {
			return
		}
	}

	return ctx.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully updated %d mappings", len(yarnMappings)),
		},
	})
}

func (m *Mappings) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "convert",
			Description: "Convert Yarn mappings to Lambda mappings",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "version",
					Description: "Mapping version",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "1.20.4",
							Value: "1.20.4",
						},
						{
							Name:  "1.20.5",
							Value: "1.20.5",
						},
						{
							Name:  "1.20.6",
							Value: "1.20.6",
						},
						{
							Name:  "1.21",
							Value: "1.21",
						},
						{
							Name:  "1.21.1",
							Value: "1.21.1",
						},
						{
							Name:  "1.21.2",
							Value: "1.21.2",
						},
						{
							Name:  "1.21.3",
							Value: "1.21.3",
						},
						{
							Name:  "1.21.4",
							Value: "1.21.4",
						},
						{
							Name:  "1.21.5",
							Value: "1.21.5",
						},
					},
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "update",
			Description: "Update all mappings on Cloudflare",
		},
	}
}

var yarnMappings = map[string]string{
	"1.20.4": "https://maven.fabricmc.net/net/fabricmc/yarn/1.20.4+build.3/yarn-1.20.4+build.3-mergedv2.jar",
	"1.20.5": "https://maven.fabricmc.net/net/fabricmc/yarn/1.20.5+build.1/yarn-1.20.5+build.1-mergedv2.jar",
	"1.20.6": "https://maven.fabricmc.net/net/fabricmc/yarn/1.20.6+build.3/yarn-1.20.6+build.3-mergedv2.jar",
	"1.21":   "https://maven.fabricmc.net/net/fabricmc/yarn/1.21+build.9/yarn-1.21+build.9-mergedv2.jar",
	"1.21.1": "https://maven.fabricmc.net/net/fabricmc/yarn/1.21.1+build.3/yarn-1.21.1+build.3-mergedv2.jar",
	"1.21.2": "https://maven.fabricmc.net/net/fabricmc/yarn/1.21.2+build.1/yarn-1.21.2+build.1-mergedv2.jar",
	"1.21.3": "https://maven.fabricmc.net/net/fabricmc/yarn/1.21.3+build.2/yarn-1.21.3+build.2-mergedv2.jar",
	"1.21.4": "https://maven.fabricmc.net/net/fabricmc/yarn/1.21.4+build.8/yarn-1.21.4+build.8-mergedv2.jar",
	"1.21.5": "https://maven.fabricmc.net/net/fabricmc/yarn/1.21.5+build.1/yarn-1.21.5+build.1-mergedv2.jar",
}

func downloadAndProcessMappings(version string) (*bytes.Buffer, error) {
	resp, err := http.Get(yarnMappings[version])
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}

	rc, err := zipReader.Open("mappings/mappings.tiny")
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return parseMappings(rc)
}

func parseMappings(rc io.Reader) (*bytes.Buffer, error) {
	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read mappings content: %w", err)
	}

	ast, errs := tinyv2parser.NewParser(bytes.NewReader(content)).Parse()
	if len(errs) > 0 {
		var errorBuffer bytes.Buffer
		for _, e := range errs {
			errorBuffer.WriteString(fmt.Sprintf("Line %d: %s\n", e.Line, e.Message))
		}
		return nil, errors.New(errorBuffer.String())
	}

	var outputBuffer bytes.Buffer
	for _, class := range ast.Classes {
		// Mappings shipped in fabric include official, intermediary and named names
		intermediate := strings.ReplaceAll(class.Names[1], "/", ".")
		named := class.Names[2][strings.LastIndex(class.Names[2], "/")+1:]

		outputBuffer.WriteString(intermediate + " " + named + "\n")

		for _, method := range class.Methods {
			outputBuffer.WriteString(method.Names[1] + " " + method.Names[2] + "\n")
		}

		for _, field := range class.Fields {
			outputBuffer.WriteString(field.Names[1] + " " + field.Names[2] + "\n")
		}
	}

	if outputBuffer.Len() > 0 {
		outputBuffer.UnreadByte()
	}

	return &outputBuffer, nil
}
