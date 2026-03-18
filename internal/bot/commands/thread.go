package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Thread command allows admins to configure auto-archive durations and lock/unlock threads.
func Thread(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "thread",
			Description:              "Thread management commands",
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(int64(discordgo.PermissionManageThreads)),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Configure thread auto-archive duration for the server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "duration",
							Description: "Auto-archive duration in minutes",
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "1 Hour", Value: 60},
								{Name: "24 Hours", Value: 1440},
								{Name: "3 Days", Value: 4320},
								{Name: "1 Week", Value: 10080},
							},
						},
					},
				},
				{
					Name:        "lock",
					Description: "Lock the current thread",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "unlock",
					Description: "Unlock the current thread",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
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
			subcommand := options[0]

			switch subcommand.Name {
			case "setup":
				duration := int(subcommand.Options[0].IntValue())
				err := database.SetThreadConfig(context.Background(), i.GuildID, duration)
				if err != nil {
					SendError(s, i, "Failed to save thread configuration.")
					return
				}
				SendEmbed(s, i, &discordgo.MessageEmbed{
					Title:       "Thread Auto-Archive Configured",
					Description: fmt.Sprintf("Threads will now automatically archive after %d minutes of inactivity.", duration),
					Color:       0x00FF00,
				})

			case "lock":
				channel, err := s.Channel(i.ChannelID)
				if err != nil {
					SendError(s, i, "Could not fetch channel info.")
					return
				}

				if channel.Type != discordgo.ChannelTypeGuildPublicThread && channel.Type != discordgo.ChannelTypeGuildPrivateThread && channel.Type != discordgo.ChannelTypeGuildNewsThread {
					SendError(s, i, "This command can only be used in a thread.")
					return
				}

				val := true
				_, err = s.ChannelEdit(i.ChannelID, &discordgo.ChannelEdit{
					Locked: &val,
				})
				if err != nil {
					SendError(s, i, "Failed to lock the thread.")
					return
				}
				SendEmbed(s, i, &discordgo.MessageEmbed{
					Description: "🔒 Thread locked.",
					Color:       0x00FF00,
				})

			case "unlock":
				channel, err := s.Channel(i.ChannelID)
				if err != nil {
					SendError(s, i, "Could not fetch channel info.")
					return
				}

				if channel.Type != discordgo.ChannelTypeGuildPublicThread && channel.Type != discordgo.ChannelTypeGuildPrivateThread && channel.Type != discordgo.ChannelTypeGuildNewsThread {
					SendError(s, i, "This command can only be used in a thread.")
					return
				}

				val := false
				_, err = s.ChannelEdit(i.ChannelID, &discordgo.ChannelEdit{
					Locked: &val,
				})
				if err != nil {
					SendError(s, i, "Failed to unlock the thread.")
					return
				}
				SendEmbed(s, i, &discordgo.MessageEmbed{
					Description: "🔓 Thread unlocked.",
					Color:       0x00FF00,
				})
			}
		},
	}
}
