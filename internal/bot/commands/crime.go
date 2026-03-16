package commands

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Crime returns a slash command to let users commit crimes for coins.
func Crime(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "crime",
			Description: "Attempt a crime to earn coins",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			guildID := i.GuildID
			var userID string
			if i.Member != nil {
				userID = i.Member.User.ID
			} else if i.User != nil {
				userID = i.User.ID
			} else {
				return
			}

			// Defer the response since db calls might take time
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			eco, err := database.GetUserEconomy(ctx, guildID, userID)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("❌ Failed to retrieve your economy data. You may need to chat a bit first!"),
				})
				return
			}

			// Check cooldown
			if eco.LastCrimeAt != nil {
				timeSince := time.Since(*eco.LastCrimeAt)
				if timeSince < 2*time.Hour {
					timeLeft := 2*time.Hour - timeSince
					minutes := int(timeLeft.Minutes())
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr(fmt.Sprintf("❌ The cops are still looking for you. Try again in %d minutes.", minutes)),
					})
					return
				}
			}

			// Perform crime activity update
			err = database.UpdateCrimeActivity(ctx, guildID, userID)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("❌ Failed to update your crime activity."),
				})
				return
			}

			embed := &discordgo.MessageEmbed{}

			// 50% success chance
			if rand.Intn(100) < 50 {
				// Success
				reward := rand.Intn(301) + 200 // 200 to 500 coins
				err = database.AddCoins(ctx, guildID, userID, reward)
				if err != nil {
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("❌ Failed to add coins to your account."),
					})
					return
				}
				embed.Title = "Crime Successful"
				embed.Description = fmt.Sprintf("You pulled off the heist and earned **%d** coins!", reward)
				embed.Color = 0x00FF00
			} else {
				// Failure
				fine := rand.Intn(151) + 50 // 50 to 200 coins fine
				err = database.RemoveCoins(ctx, guildID, userID, fine)
				if err != nil {
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("❌ Failed to deduct fine from your account."),
					})
					return
				}
				embed.Title = "Busted!"
				embed.Description = fmt.Sprintf("You were caught and fined **%d** coins.", fine)
				embed.Color = 0xFF0000
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
		},
	}
}
