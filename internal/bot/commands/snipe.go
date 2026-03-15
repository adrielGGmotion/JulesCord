package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Snipe creates a command that fetches the last deleted message in the channel.
func Snipe(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "snipe",
			Description: "Fetch the most recently deleted message in this channel",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Database is not configured.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			snipe, err := database.GetSnipe(context.Background(), i.ChannelID)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "An error occurred while trying to fetch the snipe.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			if snipe == nil {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "There is nothing to snipe in this channel.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			embed := &discordgo.MessageEmbed{
				Description: snipe.MessageContent,
				Color:       0x00FF00, // Green
				Author: &discordgo.MessageEmbedAuthor{
					Name: fmt.Sprintf("Message from <@%s>", snipe.AuthorID),
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Sniped at %s", snipe.Timestamp.Format("15:04:05")),
				},
				Timestamp: snipe.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			}

			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		},
	}
}

// EditSnipe creates a command that fetches the last edited message in the channel.
func EditSnipe(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "editsnipe",
			Description: "Fetch the most recently edited message in this channel",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Database is not configured.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			editSnipe, err := database.GetEditSnipe(context.Background(), i.ChannelID)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "An error occurred while trying to fetch the edit snipe.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			if editSnipe == nil {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "There is nothing to edit snipe in this channel.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			embed := &discordgo.MessageEmbed{
				Color: 0xFFA500, // Orange
				Author: &discordgo.MessageEmbedAuthor{
					Name: fmt.Sprintf("Message from <@%s>", editSnipe.AuthorID),
				},
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Before",
						Value:  editSnipe.OldContent,
						Inline: false,
					},
					{
						Name:   "After",
						Value:  editSnipe.NewContent,
						Inline: false,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Edit sniped at %s", editSnipe.Timestamp.Format("15:04:05")),
				},
				Timestamp: editSnipe.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			}

			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		},
	}
}
