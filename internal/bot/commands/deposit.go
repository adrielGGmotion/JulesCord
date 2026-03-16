package commands

import (
	"context"
	"fmt"
	"strconv"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Deposit returns the /deposit command definition and handler.
func Deposit(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "deposit",
			Description: "Deposit coins from your wallet into your bank",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "amount",
					Description: "The amount of coins to deposit (or 'all')",
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
			var amountToDeposit int64

			ctx := context.Background()

			// Check user's current economy balance
			econ, err := database.GetUserEconomy(ctx, i.GuildID, userID)
			if err != nil || econ == nil {
				SendError(s, i, "You don't have an economy profile yet or an error occurred.")
				return
			}

			if amountStr == "all" {
				amountToDeposit = econ.Coins
			} else {
				parsedAmount, err := strconv.ParseInt(amountStr, 10, 64)
				if err != nil || parsedAmount <= 0 {
					SendError(s, i, "Please specify a valid positive number or 'all'.")
					return
				}
				amountToDeposit = parsedAmount
			}

			if amountToDeposit <= 0 {
				SendError(s, i, "You have no coins to deposit.")
				return
			}

			if econ.Coins < amountToDeposit {
				SendError(s, i, "You do not have enough coins in your wallet.")
				return
			}

			err = database.DepositCoins(ctx, i.GuildID, userID, amountToDeposit)
			if err != nil {
				SendError(s, i, "Failed to deposit coins. Please try again.")
				return
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "🏦 Bank Deposit",
							Color:       0x2ecc71, // Green
							Description: fmt.Sprintf("Successfully deposited **%d** coins into your bank.\n\n**New Wallet:** %d\n**New Bank:** %d", amountToDeposit, econ.Coins-amountToDeposit, econ.Bank+amountToDeposit),
						},
					},
				},
			})
		},
	}
}
