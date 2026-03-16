package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// RepLB creates the /replb slash command.
func RepLB(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "replb",
			Description: "Displays the top 10 users with the highest reputation",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			topUsers, err := database.GetTopReputationUsers(ctx, i.GuildID)
			if err != nil {
				SendError(s, i, "Failed to retrieve reputation leaderboard.")
				return
			}

			if len(topUsers) == 0 {
				SendError(s, i, "Nobody has any reputation points yet.")
				return
			}

			description := ""
			for index, r := range topUsers {
				// We don't need to fetch user objects, Discord markdown <@id> will render it
				description += fmt.Sprintf("**%d.** <@%s> — %d Rep\n", index+1, r.UserID, r.Rep)
			}

			embed := &discordgo.MessageEmbed{
				Title:       "🏆 Reputation Leaderboard",
				Description: description,
				Color:       0x1ABC9C, // Teal color
				Timestamp:   time.Now().Format(time.RFC3339),
			}

			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
			if err != nil {
				_ = fmt.Errorf("failed to respond to replb interaction: %w", err)
			}
		},
	}
}
