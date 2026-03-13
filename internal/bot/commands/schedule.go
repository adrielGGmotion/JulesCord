package commands

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Schedule returns the /schedule command definition and handler.
func Schedule(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionAdministrator)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "schedule",
			Description:              "Schedule an announcement.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Adds a scheduled announcement.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send the message to.",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "minutes",
							Description: "Minutes from now to send the message.",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message",
							Description: "The message to send.",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "This command can only be used in a server.",
					},
				})
				return
			}

			if database == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Database is not connected.",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subCommand := options[0]

			if subCommand.Name == "add" {
				var channelID, message string
				var minutes int64

				for _, option := range subCommand.Options {
					switch option.Name {
					case "channel":
						channelID = option.ChannelValue(s).ID
					case "minutes":
						minutes = option.IntValue()
					case "message":
						message = option.StringValue()
					}
				}

				if minutes <= 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Minutes must be greater than 0.",
						},
					})
					return
				}

				sendAt := time.Now().Add(time.Duration(minutes) * time.Minute)

				err := database.CreateScheduledAnnouncement(context.Background(), i.GuildID, channelID, message, sendAt)
				if err != nil {
					log.Printf("Failed to create scheduled announcement: %v", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "An error occurred while scheduling the announcement.",
						},
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Announcement Scheduled",
					Description: fmt.Sprintf("Your message will be sent in <#%s> at <t:%d:f>.", channelID, sendAt.Unix()),
					Color:       0x00FF00, // Green
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			}
		},
	}
}
