package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// ServerPoints returns the serverpoints command definition.
func ServerPoints(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "serverpoints",
			Description:              "Manage custom server points for users (Admin only)",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageGuild); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add points to a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to add points to",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "The amount of points to add",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove points from a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to remove points from",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "The amount of points to remove",
							Required:    true,
						},
					},
				},
				{
					Name:        "reset",
					Description: "Reset all server points in this server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			options := i.ApplicationCommandData().Options
			subcommand := options[0].Name

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				slog.Error("Failed to defer interaction", "error", err)
				return
			}

			ctx := context.Background()

			switch subcommand {
			case "add":
				var userOpt *discordgo.User
				var amountOpt int64

				for _, opt := range options[0].Options {
					if opt.Name == "user" {
						userOpt = opt.UserValue(s)
					} else if opt.Name == "amount" {
						amountOpt = opt.IntValue()
					}
				}

				if userOpt == nil {
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("User not found."),
					})
					return
				}

				err = database.AddServerPoints(ctx, i.GuildID, userOpt.ID, amountOpt)
				if err != nil {
					slog.Error("Failed to add server points", "error", err)
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("Failed to add points due to an internal error."),
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Server Points Added",
					Description: fmt.Sprintf("Successfully added %d points to <@%s>.", amountOpt, userOpt.ID),
					Color:       0x00FF00,
				}

				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})

			case "remove":
				var userOpt *discordgo.User
				var amountOpt int64

				for _, opt := range options[0].Options {
					if opt.Name == "user" {
						userOpt = opt.UserValue(s)
					} else if opt.Name == "amount" {
						amountOpt = opt.IntValue()
					}
				}

				if userOpt == nil {
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("User not found."),
					})
					return
				}

				err = database.AddServerPoints(ctx, i.GuildID, userOpt.ID, -amountOpt)
				if err != nil {
					slog.Error("Failed to remove server points", "error", err)
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("Failed to remove points due to an internal error."),
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Server Points Removed",
					Description: fmt.Sprintf("Successfully removed %d points from <@%s>.", amountOpt, userOpt.ID),
					Color:       0xFF0000,
				}

				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})

			case "reset":
				err = database.ResetServerLeaderboard(ctx, i.GuildID)
				if err != nil {
					slog.Error("Failed to reset server points", "error", err)
					_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("Failed to reset points due to an internal error."),
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Server Points Reset",
					Description: "All server points have been successfully reset.",
					Color:       0xFFFF00,
				}

				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})
			}
		},
	}
}
