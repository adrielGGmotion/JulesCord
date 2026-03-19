package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// WelcomeRole returns the /welcomerole command.
func WelcomeRole(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "welcomerole",
			Description: "Manage welcome roles for the server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a role to be assigned when a user joins",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to assign",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a welcome role",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to remove",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all welcome roles",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageGuild),
			DMPermission:             func(b bool) *bool { return &b }(false),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			subcommand := i.ApplicationCommandData().Options[0]
			guildID := i.GuildID

			switch subcommand.Name {
			case "add":
				roleID := subcommand.Options[0].RoleValue(s, guildID).ID
				err := database.AddWelcomeRole(context.Background(), guildID, roleID)
				if err != nil {
					SendError(s, i, "Failed to add welcome role.")
					return
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully added <@&%s> to welcome roles.", roleID),
					},
				})

			case "remove":
				roleID := subcommand.Options[0].RoleValue(s, guildID).ID
				err := database.RemoveWelcomeRole(context.Background(), guildID, roleID)
				if err != nil {
					SendError(s, i, "Failed to remove welcome role.")
					return
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully removed <@&%s> from welcome roles.", roleID),
					},
				})

			case "list":
				roles, err := database.GetWelcomeRoles(context.Background(), guildID)
				if err != nil {
					SendError(s, i, "Failed to fetch welcome roles.")
					return
				}
				if len(roles) == 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "There are no welcome roles configured for this server.",
						},
					})
					return
				}

				var roleMentions []string
				for _, r := range roles {
					roleMentions = append(roleMentions, fmt.Sprintf("<@&%s>", r))
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Welcome Roles",
					Description: strings.Join(roleMentions, "\n"),
					Color:       0x00ff00,
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
