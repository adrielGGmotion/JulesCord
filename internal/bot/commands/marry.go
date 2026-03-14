package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// NewMarryCommand creates the /marry command
func NewMarryCommand(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "marry",
			Description: "Marriage system commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "propose",
					Description: "Propose marriage to someone",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to propose to",
							Required:    true,
						},
					},
				},
				{
					Name:        "accept",
					Description: "Accept a marriage proposal",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user who proposed to you",
							Required:    true,
						},
					},
				},
				{
					Name:        "divorce",
					Description: "Divorce your current partner or cancel a pending proposal",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			if i.Member == nil || i.Member.User == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "propose":
				handleMarryPropose(s, i, database, subcommand.Options)
			case "accept":
				handleMarryAccept(s, i, database, subcommand.Options)
			case "divorce":
				handleMarryDivorce(s, i, database)
			}
		},
	}
}

func handleMarryPropose(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	proposerID := i.Member.User.ID
	var proposee *discordgo.User

	for _, opt := range options {
		if opt.Name == "user" {
			proposee = opt.UserValue(s)
		}
	}

	if proposee == nil {
		SendError(s, i, "User not found.")
		return
	}

	if proposee.ID == proposerID {
		SendError(s, i, "You cannot marry yourself.")
		return
	}

	if proposee.Bot {
		SendError(s, i, "You cannot marry a bot.")
		return
	}

	err := database.ProposeMarriage(context.Background(), i.GuildID, proposerID, proposee.ID)
	if err != nil {
		slog.Error("Failed to propose marriage", "err", err)
		SendError(s, i, err.Error())
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Marriage Proposal",
		Description: fmt.Sprintf("<@%s>, you have received a marriage proposal from <@%s>! Use `/marry accept user:@%s` to accept.", proposee.ID, proposerID, i.Member.User.Username),
		Color:       0xFF69B4, // Hot Pink
	})
}

func handleMarryAccept(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	proposeeID := i.Member.User.ID
	var proposer *discordgo.User

	for _, opt := range options {
		if opt.Name == "user" {
			proposer = opt.UserValue(s)
		}
	}

	if proposer == nil {
		SendError(s, i, "User not found.")
		return
	}

	err := database.AcceptMarriage(context.Background(), i.GuildID, proposeeID, proposer.ID)
	if err != nil {
		slog.Error("Failed to accept marriage", "err", err)
		SendError(s, i, err.Error())
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Married!",
		Description: fmt.Sprintf("Congratulations! <@%s> and <@%s> are now married! 🎉💖", proposer.ID, proposeeID),
		Color:       0xFF69B4, // Hot Pink
	})
}

func handleMarryDivorce(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	userID := i.Member.User.ID

	err := database.Divorce(context.Background(), i.GuildID, userID)
	if err != nil {
		slog.Error("Failed to divorce", "err", err)
		SendError(s, i, err.Error())
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Divorced",
		Description: fmt.Sprintf("<@%s> is no longer married and has cancelled any pending proposals. 💔", userID),
		Color:       0x000000, // Black
	})
}
