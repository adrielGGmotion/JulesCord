package commands

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Work returns a slash command to let users earn coins by working.
func Work(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "work",
			Description: "Work to earn some coins",
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
			if eco.LastWorkAt != nil {
				timeSince := time.Since(*eco.LastWorkAt)
				if timeSince < time.Hour {
					timeLeft := time.Hour - timeSince
					minutes := int(timeLeft.Minutes())
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr(fmt.Sprintf("❌ You are too tired to work right now. Try again in %d minutes.", minutes)),
					})
					return
				}
			}

			err = database.UpdateWorkActivity(ctx, guildID, userID)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("❌ Failed to update your work activity."),
				})
				return
			}

			// Generate reward
			reward := rand.Intn(151) + 50 // 50 to 200 coins by default
			jobName := "hard"

			if eco.JobID != nil {
				job, err := database.GetJob(ctx, *eco.JobID)
				if err == nil {
					reward = job.Salary
					jobName = fmt.Sprintf("as a **%s**", job.Name)
				}
			}

			// Add coins and update timestamp
			err = database.AddCoins(ctx, guildID, userID, reward)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("❌ Failed to add coins to your account."),
				})
				return
			}

			embed := &discordgo.MessageEmbed{
				Title:       "Work Completed",
				Description: fmt.Sprintf("You worked %s and earned **%d** coins!", jobName, reward),
				Color:       0x00FF00,
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
		},
	}
}
