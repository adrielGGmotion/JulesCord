package commands

import (
	"context"
	"fmt"
	"strconv"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Bank returns the /bank command definition and handler.
func Bank(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "bank",
			Description: "Manage your bank account.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "balance",
					Description: "Check your bank and wallet balance.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "deposit",
					Description: "Deposit coins into your bank.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "amount",
							Description: "Amount to deposit (or 'all')",
							Required:    true,
						},
					},
				},
				{
					Name:        "withdraw",
					Description: "Withdraw coins from your bank.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "amount",
							Description: "Amount to withdraw (or 'all')",
							Required:    true,
						},
					},
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

			// Defer response since DB queries might take a moment
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0].Name
			ctx := context.Background()
			userID := ""
			if i.Member != nil && i.Member.User != nil {
				userID = i.Member.User.ID
			} else if i.User != nil {
				userID = i.User.ID
			}

			econ, err := database.GetUserEconomy(ctx, i.GuildID, userID)
			if err != nil || econ == nil {
				econ = &db.UserEconomy{Coins: 0, Bank: 0}
			}

			switch subcommand {
			case "balance":
				desc := fmt.Sprintf(
					"**Wallet:** %d coins\n**Bank:** %d coins\n**Total:** %d coins",
					econ.Coins, econ.Bank, econ.Coins+econ.Bank,
				)

				// Check for joint bank
				marriage, err := database.GetMarriage(ctx, i.GuildID, userID)
				if err == nil && marriage != nil && marriage.JointBank {
					desc += fmt.Sprintf("\n\n**Joint Bank Balance:** %d coins", marriage.JointBalance)
				}

				embed := &discordgo.MessageEmbed{
					Title: "🏦 Bank Balance",
					Color: 0x2ecc71,
					Description: desc,
				}
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})

			case "deposit":
				amountStr := options[0].Options[0].StringValue()
				var depositAmount int64

				if amountStr == "all" {
					depositAmount = econ.Coins
				} else {
					parsed, err := strconv.ParseInt(amountStr, 10, 64)
					if err != nil || parsed <= 0 {
						s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
							Content: stringPtr("Please provide a valid positive number or 'all'."),
						})
						return
					}
					depositAmount = parsed
				}

				if depositAmount <= 0 {
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("You don't have any coins to deposit!"),
					})
					return
				}

				if depositAmount > econ.Coins {
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr(fmt.Sprintf("You only have **%d** coins in your wallet.", econ.Coins)),
					})
					return
				}

				err = database.DepositCoins(ctx, i.GuildID, userID, depositAmount)
				if err != nil {
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("Failed to deposit coins. " + err.Error()),
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "🏦 Deposit Successful",
					Color:       0x2ecc71,
					Description: fmt.Sprintf("Deposited **%d** coins into your bank.\n\n**New Balances:**\nWallet: %d\nBank: %d", depositAmount, econ.Coins-depositAmount, econ.Bank+depositAmount),
				}
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})

			case "withdraw":
				amountStr := options[0].Options[0].StringValue()
				var withdrawAmount int64

				if amountStr == "all" {
					withdrawAmount = econ.Bank
				} else {
					parsed, err := strconv.ParseInt(amountStr, 10, 64)
					if err != nil || parsed <= 0 {
						s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
							Content: stringPtr("Please provide a valid positive number or 'all'."),
						})
						return
					}
					withdrawAmount = parsed
				}

				if withdrawAmount <= 0 {
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("You don't have any coins in your bank to withdraw!"),
					})
					return
				}

				if withdrawAmount > econ.Bank {
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr(fmt.Sprintf("You only have **%d** coins in your bank.", econ.Bank)),
					})
					return
				}

				err = database.WithdrawCoins(ctx, i.GuildID, userID, withdrawAmount)
				if err != nil {
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("Failed to withdraw coins. " + err.Error()),
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "🏦 Withdrawal Successful",
					Color:       0x2ecc71,
					Description: fmt.Sprintf("Withdrew **%d** coins from your bank.\n\n**New Balances:**\nWallet: %d\nBank: %d", withdrawAmount, econ.Coins+withdrawAmount, econ.Bank-withdrawAmount),
				}
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})
			}
		},
	}
}
