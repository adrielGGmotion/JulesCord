package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// WarnAutoMod returns the /warnautomod command definition and handler.
func WarnAutoMod(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionAdministrator)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "warnautomod",
			Description:              "Configure automated punishments for reaching warning thresholds.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add an automated punishment rule",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "threshold",
							Description: "Number of warnings required to trigger this action",
							Required:    true,
							MinValue:    func() *float64 { v := 1.0; return &v }(),
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "action",
							Description: "The punishment action",
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "Mute", Value: "mute"},
								{Name: "Kick", Value: "kick"},
								{Name: "Ban", Value: "ban"},
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "duration",
							Description: "Duration (for mutes or temp bans, e.g., '10m', '1h', '7d')",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove an automated punishment rule",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "threshold",
							Description: "The warning threshold to remove",
							Required:    true,
							MinValue:    func() *float64 { v := 1.0; return &v }(),
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List all automated punishment rules",
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}
			if database == nil {
				SendError(s, i, "Database is not connected.")
				return
			}

			subcommand := i.ApplicationCommandData().Options[0].Name
			options := i.ApplicationCommandData().Options[0].Options

			ctx := context.Background()

			switch subcommand {
			case "add":
				var threshold int
				var action string
				var durationStr *string

				for _, opt := range options {
					switch opt.Name {
					case "threshold":
						threshold = int(opt.IntValue())
					case "action":
						action = opt.StringValue()
					case "duration":
						val := opt.StringValue()
						durationStr = &val
					}
				}

				if action == "mute" && durationStr == nil {
					SendError(s, i, "You must specify a duration when the action is 'mute' (e.g., '10m', '1h').")
					return
				}

				if durationStr != nil {
					// Validate duration format
					parsedDurationStr := *durationStr
					if strings.HasSuffix(parsedDurationStr, "d") {
						daysStr := strings.TrimSuffix(parsedDurationStr, "d")
						days, err := strconv.Atoi(daysStr)
						if err != nil {
							SendError(s, i, "Invalid duration format. Use e.g. 10m, 1h, 1d.")
							return
						}
						parsedDurationStr = fmt.Sprintf("%dh", days*24)
					}
					_, err := time.ParseDuration(parsedDurationStr)
					if err != nil {
						SendError(s, i, "Invalid duration format. Use formats like '10m', '1h', '7d'.")
						return
					}
				}

				err := database.AddWarnAutomationRule(ctx, i.GuildID, threshold, action, durationStr)
				if err != nil {
					slog.Error("Failed to add warn automation rule", "guild_id", i.GuildID, "error", err)
					SendError(s, i, "Failed to add automated punishment rule.")
					return
				}

				desc := fmt.Sprintf("Successfully configured automated punishment: **%s** upon reaching **%d warnings**.", action, threshold)
				if durationStr != nil {
					desc = fmt.Sprintf("Successfully configured automated punishment: **%s** for **%s** upon reaching **%d warnings**.", action, *durationStr, threshold)
				}
				embed := &discordgo.MessageEmbed{
					Title:       "Warn Automation Rule Added",
					Description: desc,
					Color:       0x00FF00,
				}
				SendEmbed(s, i, embed)

			case "remove":
				var threshold int
				for _, opt := range options {
					if opt.Name == "threshold" {
						threshold = int(opt.IntValue())
					}
				}

				err := database.RemoveWarnAutomationRule(ctx, i.GuildID, threshold)
				if err != nil {
					slog.Error("Failed to remove warn automation rule", "guild_id", i.GuildID, "error", err)
					SendError(s, i, "Failed to remove automated punishment rule.")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Warn Automation Rule Removed",
					Description: fmt.Sprintf("Removed automated punishment for reaching **%d warnings**.", threshold),
					Color:       0x00FF00,
				}
				SendEmbed(s, i, embed)

			case "list":
				rules, err := database.GetWarnAutomationRules(ctx, i.GuildID)
				if err != nil {
					slog.Error("Failed to list warn automation rules", "guild_id", i.GuildID, "error", err)
					SendError(s, i, "Failed to retrieve automated punishment rules.")
					return
				}

				if len(rules) == 0 {
					SendEmbed(s, i, &discordgo.MessageEmbed{
						Title:       "Warn Automation Rules",
						Description: "No automated punishment rules configured for this server.",
						Color:       0x3498DB,
					})
					return
				}

				var desc strings.Builder
				for _, rule := range rules {
					dur := ""
					if rule.Duration != nil {
						dur = fmt.Sprintf(" (Duration: %s)", *rule.Duration)
					}
					desc.WriteString(fmt.Sprintf("**%d Warnings**: %s%s\n", rule.WarningThreshold, rule.Action, dur))
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Warn Automation Rules",
					Description: desc.String(),
					Color:       0x3498DB,
				}
				SendEmbed(s, i, embed)
			}
		},
	}
}
