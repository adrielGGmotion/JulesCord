package commands

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Coins returns the /coins command definition and handler.
func Coins(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "coins",
			Description: "Displays your or another user's coin balance.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to check the coin balance of",
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
						Content: "Database is not connected. Cannot fetch coin balance.",
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
						Content: "Bots don't have coins!",
					},
				})
				return
			}

			ctx := context.Background()

			// Fetch economy data
			econ, err := database.GetUserEconomy(ctx, i.GuildID, targetUser.ID)

			var balance int64 = 0
			if err == nil && econ != nil {
				balance = econ.Coins
			}

			// Send embed
			embed := &discordgo.MessageEmbed{
				Title: fmt.Sprintf("%s's Wallet", targetUser.Username),
				Color: 0xf1c40f, // Yellow/Gold
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targetUser.AvatarURL(""),
				},
				Description: fmt.Sprintf("💰 **%d** coins", balance),
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
