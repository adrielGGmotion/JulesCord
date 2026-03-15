package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// Report returns the /report slash command.
func Report(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "report",
			Description: "Report system commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the channel for receiving reports (Admins only)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to receive reports",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Name:        "user",
					Description: "Report a user to the moderators",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "target",
							Description: "The user to report",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "reason",
							Description: "The reason for reporting this user",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0].Name
			switch subcommand {
			case "setup":
				handleReportSetup(s, i, database)
			case "user":
				handleReportUser(s, i, database)
			}
		},
	}
}

func handleReportSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	if i.Member == nil {
		SendError(s, i, "This command can only be used in a server.")
		return
	}

	// Check permissions manually for subcommands instead of root level
	member, err := s.GuildMember(i.GuildID, i.Member.User.ID)
	if err != nil {
		SendError(s, i, "Could not verify permissions.")
		return
	}

	hasAdmin := false
	for _, roleID := range member.Roles {
		role, err := s.State.Role(i.GuildID, roleID)
		if err == nil && (role.Permissions&discordgo.PermissionAdministrator) != 0 {
			hasAdmin = true
			break
		}
	}

	// Fallback check on user permissions if state isn't populated or owner
	if !hasAdmin {
		guild, err := s.Guild(i.GuildID)
		if err == nil && guild.OwnerID == i.Member.User.ID {
			hasAdmin = true
		}
	}

	if !hasAdmin && i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		SendError(s, i, "You need Administrator permissions to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options[0].Options
	channelID := options[0].ChannelValue(s).ID

	ctx := context.Background()
	err = database.SetReportChannel(ctx, i.GuildID, channelID)
	if err != nil {
		slog.Error("Failed to set report channel", "guild", i.GuildID, "error", err)
		SendError(s, i, "Failed to set the report channel.")
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Report channel has been set to <#%s>.", channelID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("Error responding to report setup", "error", err)
	}
}

func handleReportUser(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	if i.Member == nil {
		SendError(s, i, "This command can only be used in a server.")
		return
	}

	ctx := context.Background()

	// Ensure report channel is configured
	channelID, err := database.GetReportChannel(ctx, i.GuildID)
	if err != nil {
		slog.Error("Failed to get report channel", "guild", i.GuildID, "error", err)
		SendError(s, i, "An error occurred while processing your report.")
		return
	}

	if channelID == "" {
		SendError(s, i, "The report system has not been configured in this server. Please contact an administrator.")
		return
	}

	options := i.ApplicationCommandData().Options[0].Options
	targetUser := options[0].UserValue(s)
	reason := options[1].StringValue()

	// Prevent reporting bots or self
	if targetUser.Bot {
		SendError(s, i, "You cannot report bots.")
		return
	}
	if targetUser.ID == i.Member.User.ID {
		SendError(s, i, "You cannot report yourself.")
		return
	}

	// Save report to database
	reportID, err := database.CreateReport(ctx, i.GuildID, i.Member.User.ID, targetUser.ID, reason)
	if err != nil {
		slog.Error("Failed to create report", "guild", i.GuildID, "error", err)
		SendError(s, i, "Failed to submit your report. Please try again later.")
		return
	}

	// Build the report embed
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("New Report (#%d)", reportID),
		Color: 0xff0000, // Red
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Reported User",
				Value:  fmt.Sprintf("<@%s> (%s)", targetUser.ID, targetUser.ID),
				Inline: false,
			},
			{
				Name:   "Reported By",
				Value:  fmt.Sprintf("<@%s> (%s)", i.Member.User.ID, i.Member.User.ID),
				Inline: false,
			},
			{
				Name:   "Reason",
				Value:  reason,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Review this report and take appropriate action.",
		},
	}

	// Send to report channel
	_, err = s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		slog.Error("Failed to send report to configured channel", "channel", channelID, "error", err)
		// Even if sending the embed fails, the report was logged in the DB
		SendError(s, i, "Your report was logged, but could not be sent to the mod channel. An administrator will review it.")
		return
	}

	// Acknowledge the user
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Your report against **%s** has been submitted successfully.", targetUser.Username),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("Error responding to report user", "error", err)
	}
}
