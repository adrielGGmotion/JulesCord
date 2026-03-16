package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// LevelLB creates the /levellb slash command.
func LevelLB(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "levellb",
			Description: "Displays the top 10 users with the highest level",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			topUsers, err := database.GetTopLevelUsers(ctx, i.GuildID)
			if err != nil {
				SendError(s, i, "Failed to retrieve level leaderboard.")
				return
			}

			if len(topUsers) == 0 {
				SendError(s, i, "Nobody has any levels yet.")
				return
			}

			description := ""
			for index, user := range topUsers {
				description += fmt.Sprintf("**%d.** <@%s> — Level %d (XP: %d)\n", index+1, user.UserID, user.Level, user.XP)
			}

			embed := &discordgo.MessageEmbed{
				Title:       "📈 Level Leaderboard",
				Description: description,
				Color:       0x9B59B6, // Purple
				Timestamp:   time.Now().Format(time.RFC3339),
			}

			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
			if err != nil {
				_ = fmt.Errorf("failed to respond to levellb interaction: %w", err)
			}
		},
	}
}
