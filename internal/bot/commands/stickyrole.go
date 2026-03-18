package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// StickyRole returns the /stickyrole command definition and handler.
func StickyRole(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageRoles)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "stickyrole",
			Description:              "Manage sticky roles for users.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add a sticky role to a user",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to add the sticky role to",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to make sticky",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a sticky role from a user",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to remove the sticky role from",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The sticky role to remove",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List all sticky roles for a user",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to list sticky roles for",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if len(i.ApplicationCommandData().Options) == 0 {
				return
			}

			subcommand := i.ApplicationCommandData().Options[0]
			switch subcommand.Name {
			case "add":
				userOpt := subcommand.Options[0]
				roleOpt := subcommand.Options[1]

				userID := userOpt.UserValue(s).ID
				roleID := roleOpt.Value.(string)

				err := database.SaveStickyRole(context.Background(), i.GuildID, userID, roleID)
				if err != nil {
					slog.Error("Failed to add sticky role", "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to add sticky role.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully made <@&%s> sticky for <@%s>.", roleID, userID),
					},
				})

			case "remove":
				userOpt := subcommand.Options[0]
				roleOpt := subcommand.Options[1]

				userID := userOpt.UserValue(s).ID
				roleID := roleOpt.Value.(string)

				err := database.RemoveStickyRole(context.Background(), i.GuildID, userID, roleID)
				if err != nil {
					slog.Error("Failed to remove sticky role", "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to remove sticky role.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully removed sticky status of <@&%s> for <@%s>.", roleID, userID),
					},
				})

			case "list":
				userOpt := subcommand.Options[0]
				userID := userOpt.UserValue(s).ID

				roles, err := database.GetStickyRoles(context.Background(), i.GuildID, userID)
				if err != nil {
					slog.Error("Failed to get sticky roles", "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to list sticky roles.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				if len(roles) == 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("<@%s> has no sticky roles.", userID),
						},
					})
					return
				}

				var formattedRoles []string
				for _, roleID := range roles {
					formattedRoles = append(formattedRoles, fmt.Sprintf("<@&%s>", roleID))
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Sticky roles for <@%s>: %s", userID, strings.Join(formattedRoles, ", ")),
					},
				})
			}
		},
	}
}
