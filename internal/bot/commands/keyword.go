package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func Keyword(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "keyword",
			Description: "Manage keyword notifications",
			DMPermission: func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a new keyword to notify you about",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "keyword",
							Description: "The keyword to trigger on",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a keyword notification",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "keyword",
							Description: "The keyword to remove",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List your keyword notifications",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcommand := options[0]

			ctx := context.Background()

			switch subcommand.Name {
			case "add":
				keyword := strings.ToLower(subcommand.Options[0].StringValue())
				err := database.AddKeywordNotification(ctx, i.Member.User.ID, i.GuildID, keyword)
				if err != nil {
					SendError(s, i, "Failed to add keyword notification.")
					return
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Added keyword notification for `%s`.", keyword),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			case "remove":
				keyword := strings.ToLower(subcommand.Options[0].StringValue())
				err := database.RemoveKeywordNotification(ctx, i.Member.User.ID, i.GuildID, keyword)
				if err != nil {
					SendError(s, i, "Failed to remove keyword notification.")
					return
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Removed keyword notification for `%s`.", keyword),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			case "list":
				notifs, err := database.GetKeywordNotifications(ctx, i.GuildID)
				if err != nil {
					SendError(s, i, "Failed to fetch keyword notifications.")
					return
				}
				var userKeywords []string
				for _, n := range notifs {
					if n.UserID == i.Member.User.ID {
						userKeywords = append(userKeywords, n.Keyword)
					}
				}

				var content string
				if len(userKeywords) == 0 {
					content = "You have no keyword notifications set up."
				} else {
					content = "Your keyword notifications:\n"
					for _, k := range userKeywords {
						content += fmt.Sprintf("- `%s`\n", k)
					}
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: content,
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			}
		},
	}
}
