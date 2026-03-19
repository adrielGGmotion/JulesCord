package commands

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Lottery returns the /lottery command definition and handler.
func Lottery(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "lottery",
			Description: "Economy lotteries",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a new lottery (Admin only)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "prize",
							Description: "The prize amount in coins",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "ticket_price",
							Description: "The cost per ticket",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "duration",
							Description: "Duration in hours",
							Required:    true,
						},
					},
				},
				{
					Name:        "buy",
					Description: "Buy tickets for a lottery",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "lottery_id",
							Description: "The ID of the lottery",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "Number of tickets to buy",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List active lotteries in the server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
			DMPermission: func(b bool) *bool { return &b }(false),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}
			if database == nil {
				SendError(s, i, "Database connection not available.")
				return
			}

			subcmd := i.ApplicationCommandData().Options[0]
			ctx := context.Background()

			switch subcmd.Name {
			case "create":
				perm, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
				if err != nil || perm&discordgo.PermissionManageGuild == 0 {
					SendError(s, i, "You do not have permission to use this command.")
					return
				}

				var prize, ticketPrice, duration int
				for _, opt := range subcmd.Options {
					switch opt.Name {
					case "prize":
						prize = int(opt.IntValue())
					case "ticket_price":
						ticketPrice = int(opt.IntValue())
					case "duration":
						duration = int(opt.IntValue())
					}
				}

				if prize <= 0 || ticketPrice <= 0 || duration <= 0 {
					SendError(s, i, "Prize, ticket price, and duration must be greater than zero.")
					return
				}

				endTime := time.Now().Add(time.Duration(duration) * time.Hour)
				id, err := database.CreateLottery(ctx, i.GuildID, prize, ticketPrice, endTime)
				if err != nil {
					slog.Error("Failed to create lottery", "error", err, "guild_id", i.GuildID)
					SendError(s, i, "Failed to create lottery.")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title: "🎟️ Lottery Created",
					Description: fmt.Sprintf("A new lottery has been created!\n\n**ID:** `%d`\n**Prize:** %d coins\n**Ticket Price:** %d coins\n**Ends:** <t:%d:R>", id, prize, ticketPrice, endTime.Unix()),
					Color: 0x00FF00,
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})

			case "buy":
				var lotteryID, amount int
				for _, opt := range subcmd.Options {
					switch opt.Name {
					case "lottery_id":
						lotteryID = int(opt.IntValue())
					case "amount":
						amount = int(opt.IntValue())
					}
				}

				if amount <= 0 {
					SendError(s, i, "Amount must be greater than zero.")
					return
				}

				err := database.BuyLotteryTicket(ctx, lotteryID, i.GuildID, i.Member.User.ID, amount)
				if err != nil {
					slog.Error("Failed to buy lottery ticket", "error", err, "user_id", i.Member.User.ID, "lottery_id", lotteryID)
					SendError(s, i, fmt.Sprintf("Failed to buy ticket(s): %v", err))
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Successfully purchased **%d** ticket(s) for lottery `#%d`.", amount, lotteryID),
					},
				})

			case "list":
				lotteries, err := database.GetActiveLotteries(ctx, i.GuildID)
				if err != nil {
					slog.Error("Failed to fetch active lotteries", "error", err, "guild_id", i.GuildID)
					SendError(s, i, "Failed to fetch active lotteries.")
					return
				}

				if len(lotteries) == 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "There are no active lotteries at the moment.",
						},
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title: "🎟️ Active Lotteries",
					Color: 0x00BFFF,
				}

				for _, l := range lotteries {
					embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
						Name:  fmt.Sprintf("Lottery #%d", l.ID),
						Value: fmt.Sprintf("**Prize:** %d coins\n**Ticket Price:** %d coins\n**Ends:** <t:%d:R>", l.Prize, l.TicketPrice, l.EndTime.Unix()),
						Inline: false,
					})
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
