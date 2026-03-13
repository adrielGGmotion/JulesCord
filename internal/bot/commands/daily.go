package commands

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Daily returns the /daily command definition and handler.
func Daily(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "daily",
			Description: "Claim your daily coin reward.",
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
						Content: "Database is not connected. Cannot process daily claim.",
					},
				})
				return
			}

			var user *discordgo.User
			if i.Member != nil {
				user = i.Member.User
			} else {
				user = i.User
			}

			if user == nil {
				return
			}

			ctx := context.Background()

			// Fetch current economy state
			econ, err := database.GetUserEconomy(ctx, i.GuildID, user.ID)
			if err != nil && err.Error() != "no rows in result set" {
				slog.Error("Failed to get economy for daily %s", "arg1", user.ID, "error", err)
			}

			// Check cooldown
			now := time.Now()
			if econ != nil && econ.LastDailyAt != nil {
				timeSinceLastDaily := now.Sub(*econ.LastDailyAt)
				if timeSinceLastDaily < 24*time.Hour {
					timeUntilNext := (24 * time.Hour) - timeSinceLastDaily
					hours := int(timeUntilNext.Hours())
					minutes := int(timeUntilNext.Minutes()) % 60

					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("You have already claimed your daily reward. Try again in **%d hours, %d minutes**.", hours, minutes),
						},
					})
					return
				}
			}

			// Reward amount
			amount := 100

			// Grant daily and get new total
			newCoins, err := database.ClaimDaily(ctx, i.GuildID, user.ID, amount)
			if err != nil {
				slog.Error("Failed to process daily claim for %s", "arg1", user.ID, "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "An error occurred while claiming your daily reward.",
					},
				})
				return
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("✅ You claimed your daily reward of **%d coins**!\nYour new balance is **%d coins**.", amount, newCoins),
				},
			})
		},
	}
}
