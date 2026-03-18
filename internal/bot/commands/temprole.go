package commands

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// ParseDuration handles standard duration string and specific "d" for days
func parseDuration(s string) (time.Duration, error) {
	re := regexp.MustCompile(`^(\d+)d$`)
	if match := re.FindStringSubmatch(s); match != nil {
		days, _ := strconv.Atoi(match[1])
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func TempRole(database *db.DB) *Command {
	perm := int64(discordgo.PermissionManageRoles)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "temprole",
			Description:              "Manage temporary roles",
			DefaultMemberPermissions: &perm,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Temporarily assign a role to a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The user to receive the role",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
						{
							Name:        "role",
							Description: "The role to assign",
							Type:        discordgo.ApplicationCommandOptionRole,
							Required:    true,
						},
						{
							Name:        "duration",
							Description: "Duration (e.g., 1h, 7d)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a temporary role assignment early",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The user to remove the role from",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
						{
							Name:        "role",
							Description: "The role to remove",
							Type:        discordgo.ApplicationCommandOptionRole,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcommand := options[0]

			switch subcommand.Name {
			case "add":
				userOpt := subcommand.Options[0]
				roleOpt := subcommand.Options[1]
				durationOpt := subcommand.Options[2]

				userID := userOpt.UserValue(s).ID
				roleID := roleOpt.Value.(string)
				durationStr := durationOpt.StringValue()

				duration, err := parseDuration(durationStr)
				if err != nil {
					SendError(s, i, "Invalid duration format. Use e.g. 1h, 7d.")
					return
				}

				expiresAt := time.Now().Add(duration)

				// Assign role
				err = s.GuildMemberRoleAdd(i.GuildID, userID, roleID)
				if err != nil {
					SendError(s, i, "Failed to add role. Do I have manage roles permission and is my role higher?")
					return
				}

				// Save to DB
				err = database.AddTempRole(context.Background(), i.GuildID, userID, roleID, expiresAt)
				if err != nil {
					SendError(s, i, "Role added, but failed to save temporary duration to database.")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Temporary Role Added",
					Description: fmt.Sprintf("Assigned <@&%s> to <@%s> for **%s**.\nExpires: <t:%d:R>", roleID, userID, durationStr, expiresAt.Unix()),
					Color:       0x00FF00,
				}
				SendEmbed(s, i, embed)

			case "remove":
				userOpt := subcommand.Options[0]
				roleOpt := subcommand.Options[1]

				userID := userOpt.UserValue(s).ID
				roleID := roleOpt.Value.(string)

				// Remove role
				err := s.GuildMemberRoleRemove(i.GuildID, userID, roleID)
				if err != nil {
					SendError(s, i, "Failed to remove role. Do I have manage roles permission and is my role higher?")
					return
				}

				// Clean up DB
				_ = database.RemoveTempRoleByGuildUserRole(context.Background(), i.GuildID, userID, roleID)

				embed := &discordgo.MessageEmbed{
					Title:       "Temporary Role Removed",
					Description: fmt.Sprintf("Removed <@&%s> from <@%s> and cleared temporary tracking.", roleID, userID),
					Color:       0xFF0000,
				}
				SendEmbed(s, i, embed)
			}
		},
	}
}
