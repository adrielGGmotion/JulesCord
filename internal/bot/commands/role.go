package commands

import (
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Role returns the /role command definition and handler.
func Role(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageRoles)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "role",
			Description:              "Manage user roles.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add a role to a user",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to add the role to",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to add",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a role from a user",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to remove the role from",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to remove",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "info",
					Description: "Display information about a role",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to view",
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

				err := s.GuildMemberRoleAdd(i.GuildID, userID, roleID)
				if err != nil {
					slog.Error("Failed to add role", "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to add role. Ensure the bot has permissions and its role is higher than the target role.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully added <@&%s> to <@%s>.", roleID, userID),
					},
				})

			case "remove":
				userOpt := subcommand.Options[0]
				roleOpt := subcommand.Options[1]

				userID := userOpt.UserValue(s).ID
				roleID := roleOpt.Value.(string)

				err := s.GuildMemberRoleRemove(i.GuildID, userID, roleID)
				if err != nil {
					slog.Error("Failed to remove role", "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to remove role. Ensure the bot has permissions and its role is higher than the target role.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully removed <@&%s> from <@%s>.", roleID, userID),
					},
				})

			case "info":
				roleOpt := subcommand.Options[0]
				roleID := roleOpt.Value.(string)

				role, err := s.State.Role(i.GuildID, roleID)
				if err != nil {
					roles, err := s.GuildRoles(i.GuildID)
					if err == nil {
						for _, r := range roles {
							if r.ID == roleID {
								role = r
								break
							}
						}
					}
				}

				if role == nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to fetch role info.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       fmt.Sprintf("Role Info: %s", role.Name),
					Color:       role.Color,
					Description: fmt.Sprintf("<@&%s>", role.ID),
					Fields: []*discordgo.MessageEmbedField{
						{Name: "ID", Value: role.ID, Inline: true},
						{Name: "Color", Value: fmt.Sprintf("#%06x", role.Color), Inline: true},
						{Name: "Position", Value: fmt.Sprintf("%d", role.Position), Inline: true},
						{Name: "Mentionable", Value: fmt.Sprintf("%t", role.Mentionable), Inline: true},
						{Name: "Hoisted", Value: fmt.Sprintf("%t", role.Hoist), Inline: true},
						{Name: "Managed", Value: fmt.Sprintf("%t", role.Managed), Inline: true},
					},
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
