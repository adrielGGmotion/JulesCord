package commands

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Rob returns a slash command to let users rob coins from others.
func Rob(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "rob",
			Description: "Attempt to steal coins from another user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "target",
					Description: "The user you want to rob",
					Required:    true,
				},
			},
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

			targetUser := i.ApplicationCommandData().Options[0].UserValue(s)
			if targetUser == nil {
				return
			}

			if targetUser.ID == userID {
				SendError(s, i, "You cannot rob yourself.")
				return
			}

			if targetUser.Bot {
				SendError(s, i, "You cannot rob bots.")
				return
			}

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			robberEco, err := database.GetUserEconomy(ctx, guildID, userID)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("❌ Failed to retrieve your economy data. You may need to chat a bit first!"),
				})
				return
			}

			// Check cooldown
			if robberEco.LastRobAt != nil {
				timeSince := time.Since(*robberEco.LastRobAt)
				if timeSince < 2*time.Hour {
					timeLeft := 2*time.Hour - timeSince
					minutes := int(timeLeft.Minutes())
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr(fmt.Sprintf("❌ The cops are still looking for you! Lay low for %d more minutes.", minutes)),
					})
					return
				}
			}

			victimEco, err := database.GetUserEconomy(ctx, guildID, targetUser.ID)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("❌ Failed to retrieve the target's economy data. They might not have any coins."),
				})
				return
			}

			if victimEco.Coins < 100 {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("❌ That user is too poor to rob. Pick someone with at least 100 coins."),
				})
				return
			}

			if robberEco.Coins < 250 {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("❌ You need at least 250 coins to cover potential fines if you get caught."),
				})
				return
			}

			err = database.UpdateRobActivity(ctx, guildID, userID)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("❌ Failed to update your rob activity."),
				})
				return
			}

			// 40% success rate
			success := rand.Intn(100) < 40

			if success {
				// Steal between 5% and 15% of victim's coins
				percent := float64(rand.Intn(11)+5) / 100.0
				amountToSteal := int(float64(victimEco.Coins) * percent)

				// Failsafe
				if amountToSteal <= 0 {
					amountToSteal = 1
				}
				if amountToSteal > int(victimEco.Coins) {
					amountToSteal = int(victimEco.Coins)
				}

				err = database.RobCoins(ctx, guildID, targetUser.ID, userID, int64(amountToSteal))
				if err != nil {
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("❌ An error occurred during the robbery."),
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Robbery Successful! 💰",
					Description: fmt.Sprintf("You successfully stole **%d** coins from <@%s>!", amountToSteal, targetUser.ID),
					Color:       0x00FF00,
				}
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})

			} else {
				// Caught! Fine between 5% and 10% of robber's coins, max 500, min 50. Paid to victim.
				percent := float64(rand.Intn(6)+5) / 100.0
				fine := int(float64(robberEco.Coins) * percent)

				if fine < 50 {
					fine = 50
				}
				if fine > 500 {
					fine = 500
				}
				if fine > int(robberEco.Coins) {
					fine = int(robberEco.Coins)
				}

				err = database.RobCoins(ctx, guildID, userID, targetUser.ID, int64(fine))
				if err != nil {
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("❌ An error occurred while processing your fine."),
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Busted! 🚓",
					Description: fmt.Sprintf("You got caught trying to rob <@%s>! You paid them a fine of **%d** coins.", targetUser.ID, fine),
					Color:       0xFF0000,
				}
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})
			}
		},
	}
}
