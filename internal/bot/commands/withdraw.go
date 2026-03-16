package commands

import (
	"context"
	"fmt"
	"strconv"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Withdraw returns the /withdraw command definition and handler.
func Withdraw(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "withdraw",
			Description: "Withdraw coins from your bank into your wallet",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "amount",
					Description: "The amount of coins to withdraw (or 'all')",
					Required:    true,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}
			if database == nil {
				SendError(s, i, "Database is not connected.")
				return
			}

			userID := ""
			if i.Member != nil {
				userID = i.Member.User.ID
			} else {
				userID = i.User.ID
			}

			// Get user input amount
			amountStr := i.ApplicationCommandData().Options[0].StringValue()
			var amountToWithdraw int64

			ctx := context.Background()

			// Check user's current economy balance
			econ, err := database.GetUserEconomy(ctx, i.GuildID, userID)
			if err != nil || econ == nil {
				SendError(s, i, "You don't have an economy profile yet or an error occurred.")
				return
			}

			if amountStr == "all" {
				amountToWithdraw = econ.Bank
			} else {
				parsedAmount, err := strconv.ParseInt(amountStr, 10, 64)
				if err != nil || parsedAmount <= 0 {
					SendError(s, i, "Please specify a valid positive number or 'all'.")
					return
				}
				amountToWithdraw = parsedAmount
			}

			if amountToWithdraw <= 0 {
				SendError(s, i, "You have no coins in your bank to withdraw.")
				return
			}

			if econ.Bank < amountToWithdraw {
				SendError(s, i, "You do not have enough coins in your bank.")
				return
			}

			err = database.WithdrawCoins(ctx, i.GuildID, userID, amountToWithdraw)
			if err != nil {
				SendError(s, i, "Failed to withdraw coins. Please try again.")
				return
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "🏧 Bank Withdrawal",
							Color:       0xf1c40f, // Yellow
							Description: fmt.Sprintf("Successfully withdrew **%d** coins into your wallet.\n\n**New Wallet:** %d\n**New Bank:** %d", amountToWithdraw, econ.Coins+amountToWithdraw, econ.Bank-amountToWithdraw),
						},
					},
				},
			})
		},
	}
}
