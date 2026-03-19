package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

func AutoReact(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "autoreact",
			Description: "Manage auto-react channels",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Configure auto-react for a channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to auto-react in",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "emojis",
							Description: "Comma-separated list of emojis (e.g., ✅,❌,<:custom:id>)",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove auto-react configuration for a channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to remove auto-react from",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all auto-react channels",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageServer),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database is required for this command.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "add":
				var channelID string
				var emojisStr string

				for _, opt := range subcommand.Options {
					switch opt.Name {
					case "channel":
						channelID = opt.ChannelValue(s).ID
					case "emojis":
						emojisStr = opt.StringValue()
					}
				}

				// Simple validation for custom emojis
				emojiList := strings.Split(emojisStr, ",")
				for j, emoji := range emojiList {
					emoji = strings.TrimSpace(emoji)
					if strings.HasPrefix(emoji, "<:") && strings.HasSuffix(emoji, ">") {
						emojiList[j] = strings.TrimPrefix(strings.TrimSuffix(emoji, ">"), "<:")
					} else if strings.HasPrefix(emoji, "<a:") && strings.HasSuffix(emoji, ">") {
						emojiList[j] = strings.TrimPrefix(strings.TrimSuffix(emoji, ">"), "<a:")
					} else {
						emojiList[j] = emoji
					}
				}

				cleanedEmojis := strings.Join(emojiList, ",")

				err := database.AddAutoReact(context.Background(), i.GuildID, channelID, cleanedEmojis)
				if err != nil {
					slog.Error("Failed to add auto-react config", "error", err, "guild_id", i.GuildID, "channel_id", channelID)
					SendError(s, i, "Failed to configure auto-react for the channel.")
					return
				}

				SendEmbed(s, i, &discordgo.MessageEmbed{
					Title:       "Auto-React Configured",
					Description: fmt.Sprintf("I will now automatically react to messages in <#%s> with these emojis:\n`%s`", channelID, emojisStr),
					Color:       0x00FF00,
				})

			case "remove":
				var channelID string

				for _, opt := range subcommand.Options {
					switch opt.Name {
					case "channel":
						channelID = opt.ChannelValue(s).ID
					}
				}

				err := database.RemoveAutoReact(context.Background(), i.GuildID, channelID)
				if err != nil {
					slog.Error("Failed to remove auto-react config", "error", err, "guild_id", i.GuildID, "channel_id", channelID)
					SendError(s, i, "Failed to remove auto-react configuration.")
					return
				}

				SendEmbed(s, i, &discordgo.MessageEmbed{
					Title:       "Auto-React Removed",
					Description: fmt.Sprintf("Auto-react has been removed from <#%s>.", channelID),
					Color:       0x00FF00,
				})

			case "list":
				configs, err := database.GetAutoReactChannels(context.Background(), i.GuildID)
				if err != nil {
					slog.Error("Failed to get auto-react configs", "error", err, "guild_id", i.GuildID)
					SendError(s, i, "Failed to retrieve auto-react configurations.")
					return
				}

				if len(configs) == 0 {
					SendEmbed(s, i, &discordgo.MessageEmbed{
						Title:       "Auto-React Channels",
						Description: "There are no auto-react channels configured for this server.",
						Color:       0xAAAAAA,
					})
					return
				}

				desc := ""
				for _, c := range configs {
					desc += fmt.Sprintf("<#%s>: `%s`\n", c.ChannelID, c.Emojis)
				}

				SendEmbed(s, i, &discordgo.MessageEmbed{
					Title:       "Auto-React Channels",
					Description: desc,
					Color:       0x00AFFF,
				})
			}
		},
	}
}
