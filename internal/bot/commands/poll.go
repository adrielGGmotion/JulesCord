package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"julescord/internal/db"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// PollCommand creates a poll with interactive buttons.
func PollCommand(database *db.DB) *Command {
	options := []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "question",
			Description: "The poll question",
			Required:    true,
		},
	}
	// Add up to 10 options
	for i := 1; i <= 10; i++ {
		options = append(options, &discordgo.ApplicationCommandOption{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        fmt.Sprintf("option%d", i),
			Description: fmt.Sprintf("Poll option %d", i),
			Required:    i <= 2, // Require at least 2 options
		})
	}

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "poll",
			Description: "Create a poll with interactive buttons",
			Options:     options,
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			optionsMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
			for _, opt := range i.ApplicationCommandData().Options {
				optionsMap[opt.Name] = opt
			}

			question := optionsMap["question"].StringValue()
			var pollOptions []string
			for j := 1; j <= 10; j++ {
				optName := fmt.Sprintf("option%d", j)
				if opt, ok := optionsMap[optName]; ok && opt.StringValue() != "" {
					pollOptions = append(pollOptions, opt.StringValue())
				}
			}

			if len(pollOptions) < 2 {
				SendError(s, i, "You must provide at least two options for the poll.")
				return
			}

			// Generate an ID for the poll
			pollID := fmt.Sprintf("poll_%s", i.ID) // use interaction id for uniqueness

			// Create the buttons
			var components []discordgo.MessageComponent
			var currentActionRow []discordgo.MessageComponent

			description := ""
			for idx, opt := range pollOptions {
				description += fmt.Sprintf("**%d.** %s\n\n", idx+1, opt)

				btn := discordgo.Button{
					Label:    fmt.Sprintf("%d", idx+1),
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("poll_vote_%s_%d", pollID, idx),
				}

				currentActionRow = append(currentActionRow, btn)

				// Discord allows max 5 buttons per action row
				if len(currentActionRow) == 5 || idx == len(pollOptions)-1 {
					components = append(components, discordgo.ActionsRow{
						Components: currentActionRow,
					})
					currentActionRow = nil
				}
			}

			embed := &discordgo.MessageEmbed{
				Title:       "📊 " + question,
				Description: description,
				Color:       0x3498db,
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Poll created by %s", i.Member.User.Username),
				},
			}

			// Send the response with the components
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: components,
				},
			})
			if err != nil {
				slog.Error("Failed to send poll response", "error", err)
				return
			}

			// We need the message ID to store in the DB so we can update the message later.
			// But InteractionRespond doesn't return the message ID directly. We have to fetch it.
			msg, err := s.InteractionResponse(i.Interaction)
			if err != nil {
				slog.Error("Failed to get poll message", "error", err)
				return
			}

			optionsJSON, err := json.Marshal(pollOptions)
			if err != nil {
				slog.Error("Failed to marshal poll options", "error", err)
				return
			}

			err = database.CreatePoll(context.Background(), pollID, i.GuildID, i.ChannelID, msg.ID, question, optionsJSON)
			if err != nil {
				slog.Error("Failed to store poll in DB", "error", err)
			}
		},
	}
}
