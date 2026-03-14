package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"math"
	"time"

	"github.com/bwmarrin/discordgo"
)

func Rep(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "rep",
			Description: "Reputation system commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "give",
					Description: "Give a reputation point to another user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The user to give rep to",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
					},
				},
				{
					Name:        "check",
					Description: "Check a user's reputation",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The user to check",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    false,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "give":
				handleRepGive(s, i, database, subcommand.Options)
			case "check":
				handleRepCheck(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleRepGive(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) == 0 {
		return
	}

	targetUser := options[0].UserValue(s)

	if targetUser == nil {
		SendError(s, i, "User not found.")
		return
	}

	if targetUser.ID == i.Member.User.ID {
		SendError(s, i, "You cannot give reputation to yourself.")
		return
	}

	if targetUser.Bot {
		SendError(s, i, "You cannot give reputation to bots.")
		return
	}

	guildID := i.GuildID
	senderID := i.Member.User.ID
	receiverID := targetUser.ID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	canGive, timeRemaining, err := database.CanGiveReputation(ctx, guildID, senderID)
	if err != nil {
		SendError(s, i, "Failed to check reputation status.")
		return
	}

	if !canGive {
		hours := int(timeRemaining.Hours())
		minutes := int(math.Mod(timeRemaining.Minutes(), 60))
		SendError(s, i, fmt.Sprintf("You have already given reputation recently. You can give reputation again in %dh %dm.", hours, minutes))
		return
	}

	err = database.AddReputation(ctx, guildID, senderID, receiverID)
	if err != nil {
		SendError(s, i, "Failed to give reputation.")
		return
	}

	rep, err := database.GetReputation(ctx, guildID, receiverID)
	var repText string
	if err == nil {
		repText = fmt.Sprintf("\n%s now has **%d** reputation points.", targetUser.Mention(), rep)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Reputation Given",
		Description: fmt.Sprintf("You gave a reputation point to %s!%s", targetUser.Mention(), repText),
		Color:       0x00FF00, // Green
	}

	SendEmbed(s, i, embed)
}

func handleRepCheck(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	targetUser := i.Member.User
	if len(options) > 0 && options[0].Name == "user" {
		targetUser = options[0].UserValue(s)
	}

	if targetUser == nil {
		SendError(s, i, "User not found.")
		return
	}

	guildID := i.GuildID
	userID := targetUser.ID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rep, err := database.GetReputation(ctx, guildID, userID)
	if err != nil {
		SendError(s, i, "Failed to retrieve reputation.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Reputation Check",
		Description: fmt.Sprintf("%s has **%d** reputation points.", targetUser.Mention(), rep),
		Color:       0x00BFFF, // Light blue
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: targetUser.AvatarURL(""),
		},
	}

	SendEmbed(s, i, embed)
}
