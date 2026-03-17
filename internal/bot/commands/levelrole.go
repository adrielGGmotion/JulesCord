package commands

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// LevelRole represents the /levelrole command group for managing role rewards for leveling up.
func LevelRole(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "levelrole",
			Description: "Manage level role rewards",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a role reward for a specific level",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "level",
							Description: "The level required to get the role",
							Required:    true,
							MinValue:    func(v float64) *float64 { return &v }(1),
						},
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to award",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "coins",
							Description: "The coins reward to award",
							Required:    false,
							MinValue:    func(v float64) *float64 { return &v }(0),
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a role reward for a specific level",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "level",
							Description: "The level to remove the reward from",
							Required:    true,
							MinValue:    func(v float64) *float64 { return &v }(1),
						},
					},
				},
				{
					Name:        "list",
					Description: "List all level role rewards",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageServer),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database is required for this command.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "add":
				var level int
				var roleID string
				var coinsReward int

				for _, opt := range subcommand.Options {
					switch opt.Name {
					case "level":
						level = int(opt.IntValue())
					case "role":
						roleID = opt.RoleValue(s, i.GuildID).ID
					case "coins":
						coinsReward = int(opt.IntValue())
					}
				}

				if roleID == "" {
					SendError(s, i, "Invalid role provided.")
					return
				}

				// Basic validation: Check if role exists and is manageable by bot (this is not full validation, discordgo doesn't provide easy pos check without fetching roles, but catching the ID works)
				err := database.SetLevelRole(context.Background(), i.GuildID, level, roleID, coinsReward)
				if err != nil {
					slog.Error("Failed to set level role", "error", err, "guild_id", i.GuildID, "level", level)
					SendError(s, i, "Failed to set level role reward.")
					return
				}

				desc := fmt.Sprintf("Users will now receive <@&%s> upon reaching **Level %d**.", roleID, level)
				if coinsReward > 0 {
					desc += fmt.Sprintf("\nThey will also receive **%d coins**.", coinsReward)
				}

				SendEmbed(s, i, &discordgo.MessageEmbed{
					Title:       "Level Role Added",
					Description: desc,
					Color:       0x00FF00,
				})

			case "remove":
				var level int

				for _, opt := range subcommand.Options {
					switch opt.Name {
					case "level":
						level = int(opt.IntValue())
					}
				}

				err := database.RemoveLevelRole(context.Background(), i.GuildID, level)
				if err != nil {
					slog.Error("Failed to remove level role", "error", err, "guild_id", i.GuildID, "level", level)
					SendError(s, i, "Failed to remove level role reward.")
					return
				}

				SendEmbed(s, i, &discordgo.MessageEmbed{
					Title:       "Level Role Removed",
					Description: fmt.Sprintf("Removed role reward for **Level %d**.", level),
					Color:       0x00FF00,
				})

			case "list":
				roles, err := database.GetLevelRoles(context.Background(), i.GuildID)
				if err != nil {
					slog.Error("Failed to get level roles", "error", err, "guild_id", i.GuildID)
					SendError(s, i, "Failed to retrieve level roles.")
					return
				}

				if len(roles) == 0 {
					SendEmbed(s, i, &discordgo.MessageEmbed{
						Title:       "Level Roles",
						Description: "There are no level roles configured for this server.",
						Color:       0xAAAAAA,
					})
					return
				}

				sort.Slice(roles, func(i, j int) bool {
					return roles[i].Level < roles[j].Level
				})

				description := ""
				for _, r := range roles {
					rewardStr := fmt.Sprintf("**Level %d:** <@&%s>", r.Level, r.RoleID)
					if r.CoinsReward > 0 {
						rewardStr += fmt.Sprintf(" + %d coins", r.CoinsReward)
					}
					description += rewardStr + "\n"
				}

				SendEmbed(s, i, &discordgo.MessageEmbed{
					Title:       "Level Roles",
					Description: description,
					Color:       0x00AFFF,
				})
			}
		},
	}
}
