package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Autorole returns the /autorole command definition and handler.
func Autorole(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageRoles)

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "autorole",
			Description:              "Manage the auto-role system.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the role to automatically assign to new members.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to assign.",
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

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subCommand := options[0]

			switch subCommand.Name {
			case "setup":
				handleAutoroleSetup(s, i, database, subCommand.Options)
			}
		},
	}
}

func handleAutoroleSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if database == nil {
		SendError(s, i, "Database connection not available.")
		return
	}

	var roleID string
	for _, option := range options {
		if option.Name == "role" {
			roleID = option.RoleValue(s, i.GuildID).ID
		}
	}

	if roleID == "" {
		SendError(s, i, "Invalid role provided.")
		return
	}

	err := database.SetAutoRole(context.Background(), i.GuildID, roleID)
	if err != nil {
		slog.Error("Failed to set auto-role", "guild_id", i.GuildID, "error", err)
		SendError(s, i, "Failed to set auto-role. Please try again later.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Auto-Role Configured",
		Description: fmt.Sprintf("Successfully set the auto-role to <@&%s>.\nNew members will automatically receive this role upon joining.", roleID),
		Color:       0x00FF00, // Green
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
