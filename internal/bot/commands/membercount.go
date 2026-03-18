package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// MemberCount returns the /membercount command definition and handler.
func MemberCount(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageChannels)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "membercount",
			Description:              "Configure a member count voice channel.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "setup",
					Description: "Set up the member count channel.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionChannel,
							Name:         "channel",
							Description:  "The voice channel to use",
							Required:     true,
							ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildVoice},
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "template",
							Description: "The format (default: 'Members: {count}')",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove the member count channel.",
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if len(i.ApplicationCommandData().Options) == 0 {
				return
			}

			subcommand := i.ApplicationCommandData().Options[0]
			ctx := context.Background()

			switch subcommand.Name {
			case "setup":
				var channelID string
				template := "Members: {count}"

				for _, opt := range subcommand.Options {
					switch opt.Name {
					case "channel":
						channelID = opt.Value.(string)
					case "template":
						template = opt.Value.(string)
					}
				}

				if !strings.Contains(template, "{count}") {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "The template must include `{count}`.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				err := database.SetMemberCountConfig(ctx, i.GuildID, channelID, template)
				if err != nil {
					slog.Error("Failed to save member count config", "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to save configuration.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Member count channel set to <#%s> with template `%s`. It will update shortly.", channelID, template),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})

				// Fetch member count and update channel immediately
				guild, err := s.State.Guild(i.GuildID)
				if err != nil {
					guild, err = s.Guild(i.GuildID)
				}

				if err == nil && guild != nil {
					name := strings.ReplaceAll(template, "{count}", fmt.Sprintf("%d", guild.MemberCount))
					if len(name) > 100 {
						name = name[:100] // Discord limit
					}
					_, editErr := s.ChannelEdit(channelID, &discordgo.ChannelEdit{
						Name: name,
					})
					if editErr != nil {
						slog.Error("Failed to edit member count channel", "error", editErr)
					}
				}

			case "remove":
				err := database.RemoveMemberCountConfig(ctx, i.GuildID)
				if err != nil {
					slog.Error("Failed to remove member count config", "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to remove configuration.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Member count channel tracking removed.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			}
		},
	}
}
