package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Goodbye returns the /goodbye command.
func Goodbye(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "goodbye",
			Description: "Configure goodbye messages for your server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Set the goodbye channel and message",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send goodbye messages in",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
								discordgo.ChannelTypeGuildNews,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message",
							Description: "The goodbye message. You can use {user} as a placeholder.",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove the goodbye message configuration",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			// Check for Admin permissions
			if i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
				SendError(s, i, "You need Administrator permissions to configure the goodbye system.")
				return
			}

			options := i.ApplicationCommandData().Options
			subcommand := options[0].Name

			switch subcommand {
			case "set":
				handleGoodbyeSet(s, i, database, options[0].Options)
			case "remove":
				handleGoodbyeRemove(s, i, database)
			}
		},
	}
}

func handleGoodbyeSet(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var channelID, message string
	for _, opt := range options {
		switch opt.Name {
		case "channel":
			channelID = opt.Value.(string)
		case "message":
			message = opt.StringValue()
		}
	}

	err := database.SetGoodbyeMessage(context.Background(), i.GuildID, channelID, message)
	if err != nil {
		slog.Error("Failed to set goodbye message", "guild", i.GuildID, "error", err)
		SendError(s, i, "Failed to set goodbye message.")
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Goodbye message set! It will be sent to <#%s>.", channelID),
		},
	})
	if err != nil {
		slog.Error("Failed to respond to /goodbye set", "error", err)
	}
}

func handleGoodbyeRemove(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	err := database.RemoveGoodbyeMessage(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to remove goodbye message", "guild", i.GuildID, "error", err)
		SendError(s, i, "Failed to remove goodbye message.")
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Goodbye message configuration removed.",
		},
	})
	if err != nil {
		slog.Error("Failed to respond to /goodbye remove", "error", err)
	}
}
