package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Snippet returns the /snippet slash command
func Snippet(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "snippet",
			Description: "Manage and send message snippets",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add or update a snippet",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The name of the snippet (one word)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "content",
							Description: "The content of the snippet",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a snippet",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The name of the snippet to remove",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all snippets",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "send",
					Description: "Send a snippet to the channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The name of the snippet to send",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcommand := options[0]

			guildID := i.GuildID
			if guildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			switch subcommand.Name {
			case "add":
				// Permissions check
				perm, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
				if err != nil || perm&discordgo.PermissionManageMessages == 0 {
					SendError(s, i, "You need Manage Messages permissions to add snippets.")
					return
				}

				name := strings.ToLower(subcommand.Options[0].StringValue())
				if strings.Contains(name, " ") {
					SendError(s, i, "Snippet names cannot contain spaces.")
					return
				}
				content := subcommand.Options[1].StringValue()

				err = database.AddSnippet(context.Background(), guildID, name, content)
				if err != nil {
					SendError(s, i, "Failed to add snippet.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Snippet `%s` saved successfully.", name),
					},
				})

			case "remove":
				// Permissions check
				perm, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
				if err != nil || perm&discordgo.PermissionManageMessages == 0 {
					SendError(s, i, "You need Manage Messages permissions to remove snippets.")
					return
				}

				name := strings.ToLower(subcommand.Options[0].StringValue())

				err = database.RemoveSnippet(context.Background(), guildID, name)
				if err != nil {
					SendError(s, i, "Failed to remove snippet.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Snippet `%s` removed (if it existed).", name),
					},
				})

			case "list":
				snippets, err := database.ListSnippets(context.Background(), guildID)
				if err != nil {
					SendError(s, i, "Failed to retrieve snippets.")
					return
				}

				if len(snippets) == 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "No snippets configured for this server.",
						},
					})
					return
				}

				listStr := "Available Snippets:\n- `" + strings.Join(snippets, "`\n- `") + "`"
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: listStr,
					},
				})

			case "send":
				name := strings.ToLower(subcommand.Options[0].StringValue())
				content, err := database.GetSnippet(context.Background(), guildID, name)
				if err != nil {
					SendError(s, i, "Database error.")
					return
				}

				if content == "" {
					SendError(s, i, fmt.Sprintf("Snippet `%s` not found.", name))
					return
				}

				// The bot should post the message directly to the channel and acknowledge the interaction ephemerally
				// or just reply directly to the interaction if we want the bot to just send the message

				// Acknowledging interaction and sending message to channel
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: content,
					},
				})
			}
		},
	}
}
