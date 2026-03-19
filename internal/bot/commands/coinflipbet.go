package commands

import (
	"context"
	"fmt"
	"math/rand"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// CoinflipBet returns the definition and handler for the /coinflipbet command.
func CoinflipBet(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "coinflipbet",
			Description: "Challenge another user to a coinflip bet",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "host",
					Description: "Challenge someone to a coinflip",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "opponent",
							Description: "The user to challenge",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "The amount of coins to bet",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "side",
							Description: "Heads or Tails",
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "Heads", Value: "heads"},
								{Name: "Tails", Value: "tails"},
							},
						},
					},
				},
				{
					Name:        "accept",
					Description: "Accept a coinflip challenge",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "host",
							Description: "The user who challenged you",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Type != discordgo.InteractionApplicationCommand {
				return
			}
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcommand := options[0].Name
			ctx := context.Background()

			if subcommand == "host" {
				opponentUser := options[0].Options[0].UserValue(s)
				amount := options[0].Options[1].IntValue()
				side := options[0].Options[2].StringValue()

				if opponentUser.ID == i.Member.User.ID {
					SendError(s, i, "You cannot challenge yourself.")
					return
				}
				if opponentUser.Bot {
					SendError(s, i, "You cannot challenge a bot.")
					return
				}
				if amount <= 0 {
					SendError(s, i, "Bet amount must be greater than zero.")
					return
				}

				hostEcon, err := database.GetUserEconomy(ctx, i.GuildID, i.Member.User.ID)
				if err != nil || hostEcon == nil || hostEcon.Coins < amount {
					SendError(s, i, "You do not have enough coins to make this bet.")
					return
				}

				err = database.CreateCoinflipBet(ctx, i.GuildID, i.Member.User.ID, opponentUser.ID, amount, side)
				if err != nil {
					SendError(s, i, fmt.Sprintf("Failed to create challenge: %v", err))
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Coinflip Challenge Issued! 🪙",
					Description: fmt.Sprintf("<@%s> has challenged <@%s> to a coinflip!\n\n**Bet:** %d coins\n**Host Pick:** %s\n\nUse `/coinflipbet accept host:@%s` to accept!", i.Member.User.ID, opponentUser.ID, amount, side, i.Member.User.Username),
					Color:       0xF1C40F,
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})

			} else if subcommand == "accept" {
				hostUser := options[0].Options[0].UserValue(s)

				bet, err := database.GetActiveCoinflipBet(ctx, i.GuildID, hostUser.ID, i.Member.User.ID)
				if err != nil || bet == nil {
					SendError(s, i, "No active challenge found from that user.")
					return
				}

				oppEcon, err := database.GetUserEconomy(ctx, i.GuildID, i.Member.User.ID)
				if err != nil || oppEcon == nil || oppEcon.Coins < bet.Amount {
					SendError(s, i, "You do not have enough coins to accept this bet.")
					return
				}

				hostEcon, err := database.GetUserEconomy(ctx, i.GuildID, hostUser.ID)
				if err != nil || hostEcon == nil || hostEcon.Coins < bet.Amount {
					_ = database.CancelCoinflipBet(ctx, bet.ID)
					SendError(s, i, "The host no longer has enough coins. Challenge cancelled.")
					return
				}

				// Flip here:
				result := "heads"
				if rand.Intn(2) == 1 {
					result = "tails"
				}

				var winnerID, loserID string
				if result == bet.Side {
					winnerID = bet.HostID
					loserID = bet.OpponentID
				} else {
					winnerID = bet.OpponentID
					loserID = bet.HostID
				}

				// Now call the updated AcceptCoinflipBet with the winner and loser
				err = database.AcceptCoinflipBet(ctx, bet.ID, i.GuildID, winnerID, loserID, bet.Amount)
				if err != nil {
					SendError(s, i, "An error occurred accepting the bet.")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Coinflip Result! 🪙",
					Description: fmt.Sprintf("The coin landed on **%s**!\n\n<@%s> wins **%d** coins from <@%s>!", result, winnerID, bet.Amount, loserID),
					Color:       0x00FF00,
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			}
		},
	}
}
