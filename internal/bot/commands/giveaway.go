package commands

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Giveaway creates the /giveaway command.
func Giveaway(database *db.DB) *Command {
	if database == nil {
		return nil
	}

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "giveaway",
			Description:              "Manage giveaways",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageMessages); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a new giveaway",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "prize",
							Description: "The prize for the giveaway",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "duration",
							Description: "Duration of the giveaway in minutes",
							Required:    true,
							MinValue:    func() *float64 { v := float64(1); return &v }(),
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "winners",
							Description: "Number of winners (default 1)",
							Required:    false,
							MinValue:    func() *float64 { v := float64(1); return &v }(),
						},
					},
				},
				{
					Name:        "end",
					Description: "End an active giveaway early",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message_id",
							Description: "The message ID of the giveaway",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Type != discordgo.InteractionApplicationCommand {
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0].Name
			if subcommand == "create" {
				handleGiveawayCreate(s, i, database, options[0].Options)
			} else if subcommand == "end" {
				handleGiveawayEnd(s, i, database, options[0].Options)
			}
		},
	}
}

func handleGiveawayCreate(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var prize string
	var duration int
	winnerCount := 1

	for _, opt := range options {
		if opt.Name == "prize" {
			prize = opt.StringValue()
		} else if opt.Name == "duration" {
			duration = int(opt.IntValue())
		} else if opt.Name == "winners" {
			winnerCount = int(opt.IntValue())
		}
	}

	endAt := time.Now().Add(time.Duration(duration) * time.Minute)

	embed := &discordgo.MessageEmbed{
		Title:       "🎉 **GIVEAWAY** 🎉",
		Description: fmt.Sprintf("**Prize:** %s\n**Winners:** %d\n**Ends:** <t:%d:R>\n\nReact with 🎉 to enter!", prize, winnerCount, endAt.Unix()),
		Color:       0x9B59B6, // Purple
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Ends at %s", endAt.Format(time.RFC822)),
		},
	}

	// First send a deferred response since creating the message and db entry might take a moment
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}

	// Send the giveaway message
	msg, err := s.ChannelMessageSendEmbed(i.ChannelID, embed)
	if err != nil {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Failed to create giveaway message.",
		})
		return
	}

	// Add the reaction
	err = s.MessageReactionAdd(i.ChannelID, msg.ID, "🎉")
	if err != nil {
		// Non-fatal error, but good to know
	}

	// Store in database
	err = database.CreateGiveaway(context.Background(), i.GuildID, i.ChannelID, msg.ID, prize, winnerCount, endAt)
	if err != nil {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Failed to save giveaway to database.",
		})
		return
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "Giveaway created successfully!",
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}

func handleGiveawayEnd(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var messageID string

	for _, opt := range options {
		if opt.Name == "message_id" {
			messageID = opt.StringValue()
		}
	}

	g, err := database.GetGiveawayByMessage(context.Background(), messageID)
	if err != nil {
		SendError(s, i, "Failed to retrieve giveaway.")
		return
	}

	if g == nil {
		SendError(s, i, "Giveaway not found.")
		return
	}

	if g.Ended {
		SendError(s, i, "Giveaway has already ended.")
		return
	}

	if g.GuildID != i.GuildID {
		SendError(s, i, "Giveaway belongs to a different server.")
		return
	}

	// End it immediately
	err = database.EndGiveaway(context.Background(), g.MessageID)
	if err != nil {
		SendError(s, i, "Failed to end giveaway in database.")
		return
	}

	// Send success response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Giveaway ended. Winners will be picked shortly by the background worker.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		// Log but do nothing
	}

	// We force the end_at time to be now, the background worker will pick it up on the next tick
	// Alternatively, we could pick winners here directly, but letting the worker handle it is cleaner
	// We'll update the end_at time in the DB to ensure the worker picks it up
	// To keep it simple, since we already marked it as `ended = true`, we need to change the DB logic or just pick winners here.
	// Actually, the DB worker checks `ended = false AND end_at <= NOW()`.
	// If we set `ended = true` here, the worker will ignore it. Let's pick winners here to be safe and immediate.

	PickGiveawayWinners(s, database, g)
}

// PickGiveawayWinners selects winners and announces them.
func PickGiveawayWinners(s *discordgo.Session, database *db.DB, g *db.Giveaway) {
	// First fetch entrants
	entrants, err := database.GetGiveawayEntrants(context.Background(), g.ID)
	if err != nil {
		return
	}

	// Fetch message to edit it
	msg, err := s.ChannelMessage(g.ChannelID, g.MessageID)
	if err != nil {
		return
	}

	if len(entrants) == 0 {
		_, _ = s.ChannelMessageSend(g.ChannelID, fmt.Sprintf("Giveaway ended for **%s**, but no one entered! 😢", g.Prize))

		// Update embed
		if len(msg.Embeds) > 0 {
			embed := msg.Embeds[0]
			embed.Title = "🎉 **GIVEAWAY ENDED** 🎉"
			embed.Description = fmt.Sprintf("**Prize:** %s\n\nNo valid entrants.", g.Prize)
			embed.Color = 0x95A5A6 // Gray
			_, _ = s.ChannelMessageEditEmbed(g.ChannelID, g.MessageID, embed)
		}
		return
	}

	// Shuffle entrants
	rand.Shuffle(len(entrants), func(i, j int) {
		entrants[i], entrants[j] = entrants[j], entrants[i]
	})

	// Select winners
	winnersCount := g.WinnerCount
	if winnersCount > len(entrants) {
		winnersCount = len(entrants)
	}

	winners := entrants[:winnersCount]

	winnerMentions := ""
	for _, w := range winners {
		winnerMentions += fmt.Sprintf("<@%s> ", w)
	}

	// Announce
	_, _ = s.ChannelMessageSend(g.ChannelID, fmt.Sprintf("Congratulations %s! You won the **%s**! 🎉", winnerMentions, g.Prize))

	// Update embed
	if len(msg.Embeds) > 0 {
		embed := msg.Embeds[0]
		embed.Title = "🎉 **GIVEAWAY ENDED** 🎉"
		embed.Description = fmt.Sprintf("**Prize:** %s\n**Winners:** %s", g.Prize, winnerMentions)
		embed.Color = 0x2ECC71 // Green
		_, _ = s.ChannelMessageEditEmbed(g.ChannelID, g.MessageID, embed)
	}
}
