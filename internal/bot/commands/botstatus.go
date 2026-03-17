package commands

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

func BotStatusCommand(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "botstatus",
			Description:              "Configure the bot's custom status (Bot Owner / Admin only)",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageGuild); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Set a new custom status for the bot",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "type",
							Description: "The activity type",
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "Playing", Value: int(discordgo.ActivityTypeGame)},
								{Name: "Streaming", Value: int(discordgo.ActivityTypeStreaming)},
								{Name: "Listening", Value: int(discordgo.ActivityTypeListening)},
								{Name: "Watching", Value: int(discordgo.ActivityTypeWatching)},
								{Name: "Custom", Value: int(discordgo.ActivityTypeCustom)},
								{Name: "Competing", Value: int(discordgo.ActivityTypeCompeting)},
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The status text",
							Required:    true,
						},
					},
				},
				{
					Name:        "clear",
					Description: "Clear the custom status and return to default rotation",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcommand := options[0].Name

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			if subcommand == "set" {
				var subOptions []*discordgo.ApplicationCommandInteractionDataOption
				if len(options[0].Options) > 0 {
					subOptions = options[0].Options
				}
				var activityType int
				var name string

				if subOptions != nil {
					for _, opt := range subOptions {
						switch opt.Name {
						case "type":
							activityType = int(opt.IntValue())
						case "name":
							name = opt.StringValue()
						}
					}
				}

				err = database.SetBotStatus(context.Background(), activityType, name)
				if err != nil {
					SendError(s, i, "Failed to set custom bot status")
					return
				}

				// Apply it immediately
				err = s.UpdateStatusComplex(discordgo.UpdateStatusData{
					Activities: []*discordgo.Activity{
						{
							Name: name,
							Type: discordgo.ActivityType(activityType),
						},
					},
				})
				if err != nil {
					SendError(s, i, "Saved to database, but failed to apply immediately: "+err.Error())
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "✅ Bot Status Updated",
					Description: fmt.Sprintf("Successfully updated custom status.\n**Type:** %d\n**Name:** %s", activityType, name),
					Color:       0x00FF00,
				}

				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})

			} else if subcommand == "clear" {
				err = database.DeleteBotStatus(context.Background())
				if err != nil {
					SendError(s, i, "Failed to clear custom bot status")
					return
				}

				// Apply immediately by clearing
				_ = s.UpdateGameStatus(0, "Resetting status...")

				embed := &discordgo.MessageEmbed{
					Title:       "✅ Bot Status Cleared",
					Description: "Successfully cleared custom status. The bot will return to its default rotation within 5 minutes.",
					Color:       0x00FF00,
				}

				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})
			}
		},
	}
}
