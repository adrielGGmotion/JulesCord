package commands

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Gamble returns the definition and handler for the gamble command.
func Gamble(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "gamble",
			Description: "Gamble your coins",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "coinflip",
					Description: "Bet on a coin flip",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "amount",
							Description: "Amount of coins to bet (or 'all')",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "choice",
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
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "slots",
					Description: "Play the slot machine",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "amount",
							Description: "Amount of coins to bet (or 'all')",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "stats",
					Description: "View your gambling stats",
				},
			},
			DMPermission: new(bool),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil || i.Member.User == nil {
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcmd := options[0]

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if subcmd.Name == "stats" {
				stats, err := database.GetGamblingStats(ctx, i.GuildID, i.Member.User.ID)
				if err != nil {
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to fetch gambling stats.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title: "Gambling Stats for " + i.Member.User.Username,
					Color: 0xF1C40F,
				}

				if stats == nil || stats.GamesPlayed == 0 {
					embed.Description = "You haven't played any gambling games yet."
				} else {
					winRate := float64(stats.GamesWon) / float64(stats.GamesPlayed) * 100
					embed.Fields = []*discordgo.MessageEmbedField{
						{Name: "Games Played", Value: fmt.Sprintf("%d", stats.GamesPlayed), Inline: true},
						{Name: "Games Won", Value: fmt.Sprintf("%d", stats.GamesWon), Inline: true},
						{Name: "Games Lost", Value: fmt.Sprintf("%d", stats.GamesLost), Inline: true},
						{Name: "Win Rate", Value: fmt.Sprintf("%.2f%%", winRate), Inline: true},
						{Name: "Total Coins Won", Value: fmt.Sprintf("%d", stats.CoinsWon), Inline: true},
						{Name: "Total Coins Lost", Value: fmt.Sprintf("%d", stats.CoinsLost), Inline: true},
					}
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
				return
			}

			// For coinflip and slots, get the bet amount
			if len(subcmd.Options) == 0 {
				return
			}
			amountStr := subcmd.Options[0].StringValue()

			// Fetch user's economy
			eco, err := database.GetUserEconomy(ctx, i.GuildID, i.Member.User.ID)
			if err != nil || eco == nil {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to fetch your economy data.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			var amount int
			if amountStr == "all" {
				amount = int(eco.Coins)
			} else {
				amount, err = strconv.Atoi(amountStr)
				if err != nil || amount <= 0 {
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Please provide a valid positive number or 'all'.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}
			}

			if int(eco.Coins) < amount || amount == 0 {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You don't have enough coins to make that bet.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			if subcmd.Name == "coinflip" {
				if len(subcmd.Options) < 2 {
					return
				}
				choice := subcmd.Options[1].StringValue()

				// Determine result
				result := "heads"
				if rand.Intn(2) == 1 {
					result = "tails"
				}

				won := choice == result
				var embed *discordgo.MessageEmbed

				if won {
					err = database.AddCoins(ctx, i.GuildID, i.Member.User.ID, amount)
					if err == nil {
						_ = database.UpdateGamblingStats(ctx, i.GuildID, i.Member.User.ID, amount, 0)
					}
					embed = &discordgo.MessageEmbed{
						Title:       "Coin Flip: " + result,
						Description: fmt.Sprintf("You won! You gained **%d** coins.", amount),
						Color:       0x00FF00,
					}
				} else {
					err = database.RemoveCoins(ctx, i.GuildID, i.Member.User.ID, amount)
					if err == nil {
						_ = database.UpdateGamblingStats(ctx, i.GuildID, i.Member.User.ID, 0, amount)
					}
					embed = &discordgo.MessageEmbed{
						Title:       "Coin Flip: " + result,
						Description: fmt.Sprintf("You lost! You lost **%d** coins.", amount),
						Color:       0xFF0000,
					}
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})

			} else if subcmd.Name == "slots" {
				emojis := []string{"🍎", "🍊", "🍇", "🍒", "💎", "7️⃣"}

				r1 := emojis[rand.Intn(len(emojis))]
				r2 := emojis[rand.Intn(len(emojis))]
				r3 := emojis[rand.Intn(len(emojis))]

				resultDisplay := fmt.Sprintf("[ %s | %s | %s ]", r1, r2, r3)

				won := false
				multiplier := 0

				if r1 == r2 && r2 == r3 {
					won = true
					if r1 == "7️⃣" {
						multiplier = 10
					} else if r1 == "💎" {
						multiplier = 5
					} else {
						multiplier = 3
					}
				} else if r1 == r2 || r2 == r3 || r1 == r3 {
					won = true
					multiplier = 2
				}

				var embed *discordgo.MessageEmbed
				if won {
					winnings := amount * multiplier
					netWin := winnings - amount
					err = database.AddCoins(ctx, i.GuildID, i.Member.User.ID, netWin)
					if err == nil {
						_ = database.UpdateGamblingStats(ctx, i.GuildID, i.Member.User.ID, netWin, 0)
					}
					embed = &discordgo.MessageEmbed{
						Title:       "Slots",
						Description: fmt.Sprintf("%s\n\nYou won! Winnings: **%d** coins.", resultDisplay, winnings),
						Color:       0x00FF00,
					}
				} else {
					err = database.RemoveCoins(ctx, i.GuildID, i.Member.User.ID, amount)
					if err == nil {
						_ = database.UpdateGamblingStats(ctx, i.GuildID, i.Member.User.ID, 0, amount)
					}
					embed = &discordgo.MessageEmbed{
						Title:       "Slots",
						Description: fmt.Sprintf("%s\n\nYou lost! You lost **%d** coins.", resultDisplay, amount),
						Color:       0xFF0000,
					}
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			}
		},
	}
}
