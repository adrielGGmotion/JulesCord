package commands

import (
	"context"
	"fmt"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// LevelBlacklist returns the /levelblacklist command.
func LevelBlacklist(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageGuild)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "levelblacklist",
			Description:              "Manage the leveling role blacklist",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a role to the leveling blacklist",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to blacklist from earning XP",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a role from the leveling blacklist",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to remove from the blacklist",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all blacklisted roles",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
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

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcmd := options[0]

			switch subcmd.Name {
			case "add":
				roleID := subcmd.Options[0].Value.(string)
				err := database.AddLevelingBlacklist(context.Background(), i.GuildID, roleID)
				if err != nil {
					SendError(s, i, "Failed to add role to the blacklist.")
					return
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Role <@&%s> has been added to the leveling blacklist. Members with this role will no longer earn XP.", roleID),
					},
				})

			case "remove":
				roleID := subcmd.Options[0].Value.(string)
				err := database.RemoveLevelingBlacklist(context.Background(), i.GuildID, roleID)
				if err != nil {
					SendError(s, i, "Failed to remove role from the blacklist.")
					return
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Role <@&%s> has been removed from the leveling blacklist.", roleID),
					},
				})

			case "list":
				roleIDs, err := database.GetLevelingBlacklists(context.Background(), i.GuildID)
				if err != nil {
					SendError(s, i, "Failed to fetch leveling blacklist.")
					return
				}
				if len(roleIDs) == 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "ℹ️ The leveling blacklist is currently empty.",
						},
					})
					return
				}

				var mentions []string
				for _, id := range roleIDs {
					mentions = append(mentions, fmt.Sprintf("• <@&%s>", id))
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Leveling Blacklist",
					Description: "Members with the following roles will not earn XP:\n" + strings.Join(mentions, "\n"),
					Color:       0x2b2d31,
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
