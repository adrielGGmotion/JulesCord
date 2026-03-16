package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// Transfer returns the /transfer command definition and handler.
func Transfer(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "transfer",
			Description: "Send coins to another user.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to send coins to",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "amount",
					Description: "The amount of coins to send",
					Required:    true,
					MinValue:    func() *float64 { v := 1.0; return &v }(),
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
				SendError(s, i, "Database is not connected.")
				return
			}

			var targetUser *discordgo.User
			var amount int64

			for _, option := range i.ApplicationCommandData().Options {
				switch option.Name {
				case "user":
					targetUser = option.UserValue(s)
				case "amount":
					amount = int64(option.IntValue())
				}
			}

			if targetUser == nil || amount <= 0 {
				SendError(s, i, "Invalid target user or amount.")
				return
			}

			var sender *discordgo.User
			if i.Member != nil {
				sender = i.Member.User
			} else {
				sender = i.User
			}

			if sender.ID == targetUser.ID {
				SendError(s, i, "You cannot send coins to yourself.")
				return
			}

			if targetUser.Bot {
				SendError(s, i, "Bots do not have coin balances.")
				return
			}

			// Defer response since DB operation might take a moment
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})

			ctx := context.Background()

			err := database.TransferCoins(ctx, i.GuildID, sender.ID, targetUser.ID, amount)
			if err != nil {
				if err.Error() == "insufficient funds or user not found" {
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("You do not have enough coins to complete this transfer."),
					})
				} else {
					slog.Error("Error transferring coins", "error", err)
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("An error occurred while transferring coins."),
					})
				}
				return
			}

			embed := &discordgo.MessageEmbed{
				Title:       "Transfer Successful",
				Description: fmt.Sprintf("**%s** sent **%d** coins to **%s**! 💸", sender.Username, amount, targetUser.Username),
				Color:       0x2ecc71, // Green
			}

			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
		},
	}
}
