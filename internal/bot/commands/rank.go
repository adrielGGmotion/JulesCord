package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Rank returns the /rank command definition and handler.
func Rank(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "rank",
			Description: "Displays your or another user's XP, level, and server rank.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to check the rank of",
					Required:    false,
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
						Content: "Database is not connected. Cannot fetch rank data.",
					},
				})
				return
			}

			// Determine target user
			var targetUser *discordgo.User
			for _, option := range i.ApplicationCommandData().Options {
				if option.Name == "user" {
					targetUser = option.UserValue(s)
				}
			}

			if targetUser == nil {
				if i.Member != nil {
					targetUser = i.Member.User
				} else {
					targetUser = i.User
				}
			}

			if targetUser.Bot {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Bots don't earn XP!",
					},
				})
				return
			}

			ctx := context.Background()

			// Fetch economy data
			econ, err := database.GetUserEconomy(ctx, i.GuildID, targetUser.ID)
			if err != nil || econ == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "User has no XP yet.",
					},
				})
				return
			}

			// Fetch rank
			rank, err := database.GetRank(ctx, i.GuildID, targetUser.ID)
			if err != nil {
				log.Printf("Failed to get rank for user %s: %v", targetUser.ID, err)
				rank = 0 // Default to 0 if error
			}

			// Send embed
			embed := &discordgo.MessageEmbed{
				Title: fmt.Sprintf("%s's Rank", targetUser.Username),
				Color: 0x3498db, // Blue
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targetUser.AvatarURL(""),
				},
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Rank",
						Value:  fmt.Sprintf("#%d", rank),
						Inline: true,
					},
					{
						Name:   "Level",
						Value:  fmt.Sprintf("%d", econ.Level),
						Inline: true,
					},
					{
						Name:   "XP",
						Value:  fmt.Sprintf("%d", econ.XP),
						Inline: true,
					},
				},
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		},
	}
}
