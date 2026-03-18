package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Cooldown command to set custom cooldowns for users
func Cooldown(database *db.DB) *Command {
	perm := int64(discordgo.PermissionManageServer)

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "cooldown",
			Description:              "Set a custom command cooldown for a user",
			DefaultMemberPermissions: &perm,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to apply the cooldown to",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "command",
					Description: "The command name (without /)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "duration",
					Description: "The cooldown duration (e.g., 5m, 1h, 1d) or 0s to clear",
					Required:    true,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database is not configured.")
				return
			}

			options := i.ApplicationCommandData().Options
			var targetUser *discordgo.User
			var cmdName string
			var durationStr string

			for _, opt := range options {
				switch opt.Name {
				case "user":
					targetUser = opt.UserValue(s)
				case "command":
					cmdName = opt.StringValue()
				case "duration":
					durationStr = opt.StringValue()
				}
			}

			// Parse duration
			var duration time.Duration
			var err error

			// Handle 'd' for days
			if len(durationStr) > 1 && durationStr[len(durationStr)-1] == 'd' {
				var days int
				_, err = fmt.Sscanf(durationStr, "%dd", &days)
				if err == nil {
					duration = time.Duration(days) * 24 * time.Hour
				}
			} else {
				duration, err = time.ParseDuration(durationStr)
			}

			if err != nil {
				SendError(s, i, "Invalid duration format. Use e.g., `5m`, `1h`, `1d`.")
				return
			}

			err = database.SetCommandCooldown(context.Background(), targetUser.ID, cmdName, duration)
			if err != nil {
				SendError(s, i, "Failed to set command cooldown.")
				return
			}

			msg := fmt.Sprintf("Set cooldown of %s for user %s on command `/%s`.", duration.String(), targetUser.Mention(), cmdName)
			if duration == 0 {
				msg = fmt.Sprintf("Cleared cooldown for user %s on command `/%s`.", targetUser.Mention(), cmdName)
			}

			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
				},
			})
		},
	}
}
