package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Modmail defines the /modmail command and its subcommands.
func Modmail(database *db.DB) *Command {
	var defaultMemberPermissions int64 = discordgo.PermissionAdministrator

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "modmail",
			Description: "ModMail System configuration",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Configure the ModMail category and log channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "category",
							Description: "The category where ModMail threads will be created",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildCategory,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "log_channel",
							Description: "The channel where ModMail logs will be sent",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Name:        "close",
					Description: "Close the current ModMail thread",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
			DefaultMemberPermissions: &defaultMemberPermissions,
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0].Name
			switch subcommand {
			case "setup":
				handleModmailSetup(s, i, database, options[0].Options)
			case "close":
				handleModmailClose(s, i, database)
			}
		},
	}
}

func handleModmailSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		SendError(s, i, "You need Administrator permissions to use this command.")
		return
	}

	var categoryID, logChannelID string
	for _, opt := range options {
		if opt.Name == "category" {
			categoryID = opt.ChannelValue(nil).ID
		} else if opt.Name == "log_channel" {
			logChannelID = opt.ChannelValue(nil).ID
		}
	}

	err := database.SetModmailConfig(context.Background(), i.GuildID, categoryID, logChannelID)
	if err != nil {
		SendError(s, i, "Failed to configure ModMail.")
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "✅ ModMail Configured",
		Description: fmt.Sprintf("Category set to <#%s>.\nLog channel set to <#%s>.", categoryID, logChannelID),
		Color:       0x00FF00,
	})
}

func handleModmailClose(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	thread, err := database.GetModmailThreadByChannel(context.Background(), i.ChannelID)
	if err != nil {
		SendError(s, i, "Failed to fetch thread info.")
		return
	}
	if thread == nil || !thread.IsOpen {
		SendError(s, i, "This command can only be used in an open ModMail thread channel.")
		return
	}

	// Defer response to avoid interaction failed
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	err = database.CloseModmailThread(context.Background(), i.ChannelID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("❌ Failed to close thread in database."),
		})
		return
	}

	dmChannel, dmErr := s.UserChannelCreate(thread.UserID)
	if dmErr == nil {
		s.ChannelMessageSendEmbed(dmChannel.ID, &discordgo.MessageEmbed{
			Title:       "ModMail Closed",
			Description: "Your ModMail thread has been closed by a moderator.",
			Color:       0xFF0000,
		})
	}

	s.ChannelDelete(i.ChannelID)

	config, err := database.GetModmailConfig(context.Background(), thread.GuildID)
	if err == nil && config != nil && config.LogChannelID != "" {
		s.ChannelMessageSendEmbed(config.LogChannelID, &discordgo.MessageEmbed{
			Title:       "🔒 ModMail Thread Closed",
			Description: fmt.Sprintf("Thread for <@%s> closed by <@%s>.", thread.UserID, i.Member.User.ID),
			Color:       0xFF0000,
		})
	}
}
