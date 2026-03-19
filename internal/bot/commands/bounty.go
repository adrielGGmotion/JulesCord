package commands

import (
	"context"
	"fmt"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Bounty returns a slash command to let users place bounties on others.
func Bounty(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "bounty",
			Description: "Manage economy bounties",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "place",
					Description: "Place a bounty on a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "target",
							Description: "The user to place a bounty on",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "The amount of coins for the bounty",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List active bounties",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "remove",
					Description: "Remove a bounty you placed on a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "target",
							Description: "The user you placed a bounty on",
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

			if subcommand == "place" {
				targetUser := options[0].Options[0].UserValue(s)
				amount := options[0].Options[1].IntValue()

				if targetUser.ID == i.Member.User.ID {
					SendError(s, i, "You cannot place a bounty on yourself.")
					return
				}

				if targetUser.Bot {
					SendError(s, i, "You cannot place a bounty on bots.")
					return
				}

				if amount <= 0 {
					SendError(s, i, "Bounty amount must be greater than zero.")
					return
				}

				err := database.PlaceBounty(ctx, i.GuildID, targetUser.ID, i.Member.User.ID, amount)
				if err != nil {
					SendError(s, i, fmt.Sprintf("Failed to place bounty: %v", err))
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Bounty Placed! 🎯",
					Description: fmt.Sprintf("**%s** placed a bounty of **%d** coins on <@%s>!", i.Member.User.Username, amount, targetUser.ID),
					Color:       0xFF0000,
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			} else if subcommand == "list" {
				bounties, err := database.GetActiveBounties(ctx, i.GuildID)
				if err != nil {
					SendError(s, i, "Failed to fetch active bounties.")
					return
				}

				if len(bounties) == 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "There are no active bounties in this server.",
						},
					})
					return
				}

				var sb strings.Builder
				for index, b := range bounties {
					sb.WriteString(fmt.Sprintf("**%d.** <@%s> - **%d** coins (Placed by <@%s>)\n", index+1, b.TargetUserID, b.BountyAmount, b.CreatedBy))
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Active Bounties 🎯",
					Description: sb.String(),
					Color:       0xF1C40F,
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})

			} else if subcommand == "remove" {
				targetUser := options[0].Options[0].UserValue(s)

				// First, check if the bounty exists and if the user who placed it is the one removing it
				bounty, err := database.GetBounty(ctx, i.GuildID, targetUser.ID)
				if err != nil {
					SendError(s, i, "An error occurred while fetching the bounty.")
					return
				}

				if bounty == nil {
					SendError(s, i, "No active bounty found for this user.")
					return
				}

				if bounty.CreatedBy != i.Member.User.ID {
					SendError(s, i, "You can only remove bounties that you have placed.")
					return
				}

				err = database.RemoveBounty(ctx, i.GuildID, targetUser.ID)
				if err != nil {
					SendError(s, i, "Failed to remove the bounty.")
					return
				}

				// Refund the coins
				err = database.AddCoins(ctx, i.GuildID, i.Member.User.ID, int(bounty.BountyAmount))
				if err != nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "The bounty was removed, but an error occurred refunding your coins. Please contact an admin.",
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully removed the bounty on <@%s>. **%d** coins have been refunded to you.", targetUser.ID, bounty.BountyAmount),
					},
				})
			}
		},
	}
}
