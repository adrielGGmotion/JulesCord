package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

var numberEmojis = []string{
	"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟",
}

// Poll returns the definition and handler for the /poll command.
func Poll(database *db.DB) *Command {
	options := []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "question",
			Description: "The question for the poll",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "option1",
			Description: "First option",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "option2",
			Description: "Second option",
			Required:    true,
		},
	}

	for i := 3; i <= 10; i++ {
		options = append(options, &discordgo.ApplicationCommandOption{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        fmt.Sprintf("option%d", i),
			Description: fmt.Sprintf("Option %d", i),
			Required:    false,
		})
	}

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "poll",
			Description: "Manage polls",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a new poll",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options:     options,
				},
				{
					Name:        "close",
					Description: "Close an existing poll and show results",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message_id",
							Description: "The message ID of the poll to close",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			subcommand := i.ApplicationCommandData().Options[0]
			switch subcommand.Name {
			case "create":
				handlePollCreate(s, i, database, subcommand.Options)
			case "close":
				handlePollClose(s, i, database, subcommand.Options)
			}
		},
	}
}

func handlePollCreate(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	optsMap := make(map[string]string)
	for _, opt := range options {
		optsMap[opt.Name] = opt.StringValue()
	}

	question := optsMap["question"]

	var pollOptions []string
	for idx := 1; idx <= 10; idx++ {
		optName := fmt.Sprintf("option%d", idx)
		if val, ok := optsMap[optName]; ok && val != "" {
			pollOptions = append(pollOptions, val)
		}
	}

	var descriptionBuilder strings.Builder
	for idx, opt := range pollOptions {
		descriptionBuilder.WriteString(fmt.Sprintf("%s %s\n", numberEmojis[idx], opt))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📊 Poll: " + question,
		Description: descriptionBuilder.String(),
		Color:       0x3498db, // Blue
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Poll created by %s", i.Member.User.String()),
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		slog.Error("Failed to respond to poll create interaction", "error", err)
		return
	}

	// Fetch the message that was just sent so we can get its ID and add reactions
	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		slog.Error("Failed to fetch poll message", "error", err)
		return
	}

	for idx := range pollOptions {
		err = s.MessageReactionAdd(i.ChannelID, msg.ID, numberEmojis[idx])
		if err != nil {
			slog.Error("Failed to add poll reaction", "error", err, "emoji", numberEmojis[idx])
		}
	}

	poll := &db.Poll{
		GuildID:   i.GuildID,
		ChannelID: i.ChannelID,
		MessageID: msg.ID,
		CreatorID: i.Member.User.ID,
		Question:  question,
		Options:   pollOptions,
	}

	err = database.CreatePoll(context.Background(), poll)
	if err != nil {
		slog.Error("Failed to save poll to database", "error", err)
	}
}

func handlePollClose(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	messageID := options[0].StringValue()

	poll, err := database.GetPoll(context.Background(), messageID)
	if err != nil {
		slog.Error("Failed to fetch poll from database", "error", err)
		SendError(s, i, "Failed to retrieve poll information.")
		return
	}
	if poll == nil {
		SendError(s, i, "Poll not found. Please provide a valid poll message ID.")
		return
	}

	if poll.IsClosed {
		SendError(s, i, "This poll is already closed.")
		return
	}

	// Check permissions: only creator or admin can close
	isAdmin := false
	if i.Member.Permissions&discordgo.PermissionAdministrator != 0 {
		isAdmin = true
	}
	if poll.CreatorID != i.Member.User.ID && !isAdmin {
		SendError(s, i, "You do not have permission to close this poll. Only the creator or an administrator can close it.")
		return
	}

	// Fetch the message from Discord to count reactions
	msg, err := s.ChannelMessage(poll.ChannelID, poll.MessageID)
	if err != nil {
		slog.Error("Failed to fetch poll message from Discord", "error", err)
		SendError(s, i, "Failed to fetch the poll message from Discord. It might have been deleted.")
		return
	}

	// Tally results
	results := make(map[string]int)
	for _, reaction := range msg.Reactions {
		for idx, emoji := range numberEmojis {
			if idx >= len(poll.Options) {
				break
			}
			if reaction.Emoji.Name == emoji {
				// Subtract 1 because the bot added the initial reaction
				count := reaction.Count - 1
				if count < 0 {
					count = 0
				}
				results[poll.Options[idx]] = count
				break
			}
		}
	}

	// Build results description
	var descriptionBuilder strings.Builder
	descriptionBuilder.WriteString("**Final Results:**\n\n")
	totalVotes := 0
	for _, count := range results {
		totalVotes += count
	}

	for idx, opt := range poll.Options {
		count := results[opt]
		percentage := 0.0
		if totalVotes > 0 {
			percentage = float64(count) / float64(totalVotes) * 100
		}
		descriptionBuilder.WriteString(fmt.Sprintf("%s **%s**: %d votes (%.1f%%)\n", numberEmojis[idx], opt, count, percentage))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📊 Poll Closed: " + poll.Question,
		Description: descriptionBuilder.String(),
		Color:       0x95a5a6, // Gray
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Poll closed by %s • Total votes: %d", i.Member.User.String(), totalVotes),
		},
	}

	// Acknowledge the interaction first
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Poll has been closed. Updating the original message with results.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("Failed to respond to poll close interaction", "error", err)
	}

	// Update the original message
	_, err = s.ChannelMessageEditEmbed(poll.ChannelID, poll.MessageID, embed)
	if err != nil {
		slog.Error("Failed to edit poll message with results", "error", err)
	}

	// Try to remove all reactions, ignoring errors if we don't have permission
	_ = s.MessageReactionsRemoveAll(poll.ChannelID, poll.MessageID)

	// Update database
	err = database.ClosePoll(context.Background(), poll.MessageID)
	if err != nil {
		slog.Error("Failed to mark poll as closed in database", "error", err)
	}
}
