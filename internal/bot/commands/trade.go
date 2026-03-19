package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Trade returns the trade command definition.
func Trade(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "trade",
			Description: "Trade coins with other users",
			DMPermission: func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "offer",
					Description: "Offer a trade to a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The user to trade with",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
						{
							Name:        "give",
							Description: "Amount of coins to give",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
							MinValue:    func(i float64) *float64 { return &i }(0),
						},
						{
							Name:        "receive",
							Description: "Amount of coins to receive",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
							MinValue:    func(i float64) *float64 { return &i }(0),
						},
					},
				},
				{
					Name:        "accept",
					Description: "Accept a pending trade",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "trade_id",
							Description: "The ID of the trade to accept",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			subcommand := options[0].Name
			guildID := i.GuildID
			userID := i.Member.User.ID

			if subcommand == "offer" {
				subOptions := options[0].Options
				var targetUser *discordgo.User
				var giveAmount, receiveAmount int

				for _, opt := range subOptions {
					switch opt.Name {
					case "user":
						targetUser = opt.UserValue(s)
					case "give":
						giveAmount = int(opt.IntValue())
					case "receive":
						receiveAmount = int(opt.IntValue())
					}
				}

				if targetUser.Bot || targetUser.ID == userID {
					SendError(s, i, "You cannot trade with a bot or yourself.")
					return
				}

				if giveAmount == 0 && receiveAmount == 0 {
					SendError(s, i, "Trade must involve some coins.")
					return
				}

				tradeID, err := database.CreateTrade(context.Background(), guildID, userID, targetUser.ID, giveAmount, receiveAmount)
				if err != nil {
					SendError(s, i, fmt.Sprintf("Failed to create trade: %v", err))
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Trade offer created (ID: %d). <@%s>, use `/trade accept trade_id:%d` to accept. You give %d, and they give %d.", tradeID, targetUser.ID, tradeID, receiveAmount, giveAmount),
					},
				})
			} else if subcommand == "accept" {
				subOptions := options[0].Options
				tradeID := int(subOptions[0].IntValue())

				err := database.AcceptTrade(context.Background(), tradeID, guildID, userID)
				if err != nil {
					SendError(s, i, fmt.Sprintf("Failed to accept trade: %v", err))
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully accepted trade #%d!", tradeID),
					},
				})
			}
		},
	}
}
