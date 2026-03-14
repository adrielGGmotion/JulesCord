package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// AFKCommand sets a user's AFK status.
func AFKCommand(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "afk",
			Description: "Set your AFK status. Removes automatically when you type.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "Reason for being AFK",
					Required:    false,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			reason := "AFK"
			options := i.ApplicationCommandData().Options
			if len(options) > 0 {
				reason = options[0].StringValue()
			}

			userID := i.Member.User.ID
			guildID := i.GuildID

			err := database.SetAFK(context.Background(), userID, guildID, reason)
			if err != nil {
				SendError(s, i, "Failed to set AFK status. Please try again.")
				return
			}

			SendEmbed(s, i, &discordgo.MessageEmbed{
				Title:       "AFK Status Set",
				Description: fmt.Sprintf("%s, I set your AFK: %s", i.Member.Mention(), reason),
				Color:       0x00FF00,
			})
		},
	}
}