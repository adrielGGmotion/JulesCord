package commands

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// AntiSpam returns the /antispam command definition and handler.
func AntiSpam(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionAdministrator)
	minValue := float64(1)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "antispam",
			Description:              "Configure the anti-spam system.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Configure anti-spam limits and actions.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "limit",
							Description: "Number of messages allowed within the time window.",
							Required:    true,
							MinValue:    &minValue,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "window",
							Description: "Time window in seconds to track messages.",
							Required:    true,
							MinValue:    &minValue,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "mute_duration",
							Description: "Duration to mute spammers (e.g. 10m, 1h).",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "This command can only be used in a server.",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]
			if subcommand.Name == "setup" {
				var limit, window int
				var muteDuration string

				for _, opt := range subcommand.Options {
					switch opt.Name {
					case "limit":
						limit = int(opt.IntValue())
					case "window":
						window = int(opt.IntValue())
					case "mute_duration":
						muteDuration = opt.StringValue()
					}
				}

				// Validate mute duration
				// Convert days (e.g., "7d") to hours since time.ParseDuration doesn't support 'd' natively
				if len(muteDuration) > 1 && muteDuration[len(muteDuration)-1] == 'd' {
					daysStr := muteDuration[:len(muteDuration)-1]
					var days int
					fmt.Sscanf(daysStr, "%d", &days)
					muteDuration = fmt.Sprintf("%dh", days*24)
				}
				_, err := time.ParseDuration(muteDuration)
				if err != nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Invalid duration format. Please use formats like `10m`, `1h`, or `1d`.",
						},
					})
					return
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err = database.SetAntiSpamConfig(ctx, i.GuildID, limit, window, muteDuration)
				if err != nil {
					slog.Error("Failed to set anti-spam config", "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "There was an error updating the anti-spam configuration.",
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Anti-spam configured! Users who send **%d** messages within **%d** seconds will be muted for **%s**.", limit, window, muteDuration),
					},
				})
			}
		},
	}
}
