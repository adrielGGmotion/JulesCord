package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// NewMemberCountCommand creates the /membercount command
func NewMemberCountCommand(database *db.DB) *Command {
	perm := int64(discordgo.PermissionManageChannels)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "membercount",
			Description:              "Configure a channel to display the server's member count",
			DefaultMemberPermissions: &perm,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set up the member count channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to display the member count (usually a Voice Channel)",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove the member count channel configuration",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subCommand := options[0]

			switch subCommand.Name {
			case "setup":
				handleMemberCountSetup(s, i, database, subCommand.Options)
			case "remove":
				handleMemberCountRemove(s, i, database)
			}
		},
	}
}

func handleMemberCountSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) == 0 {
		SendError(s, i, "Missing channel option.")
		return
	}

	channelID := options[0].Value.(string)

	err := database.SetMemberCountChannel(context.Background(), i.GuildID, channelID)
	if err != nil {
		slog.Error("Failed to set member count channel", "error", err)
		SendError(s, i, "Failed to set the member count channel.")
		return
	}

	// Try to get the guild from state
	guild, err := s.State.Guild(i.GuildID)
	if err != nil || guild == nil {
		guild, err = s.Guild(i.GuildID)
		if err != nil {
			slog.Error("Failed to fetch guild for member count", "error", err)
		}
	}

	memberCount := 0
	if guild != nil {
		memberCount = guild.MemberCount
		if memberCount == 0 {
			memberCount = len(guild.Members)
		}

		if memberCount > 0 {
			newName := fmt.Sprintf("Members: %d", memberCount)
			_, err = s.ChannelEdit(channelID, &discordgo.ChannelEdit{Name: newName})
			if err != nil {
				slog.Error("Failed to update channel name in member count setup", "error", err)
			}
		}
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully configured <#%s> as the member count channel.", channelID),
		},
	})
	if err != nil {
		slog.Error("Failed to send membercount setup response", "error", err)
	}
}

func handleMemberCountRemove(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	err := database.RemoveMemberCountChannel(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to remove member count channel", "error", err)
		SendError(s, i, "Failed to remove the member count channel configuration.")
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Successfully removed the member count channel configuration.",
		},
	})
	if err != nil {
		slog.Error("Failed to send membercount remove response", "error", err)
	}
}
