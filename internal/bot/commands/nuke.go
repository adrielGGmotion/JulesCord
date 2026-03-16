package commands

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

func Nuke(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "nuke",
			Description:              "Nukes the current channel by deleting it and recreating a clean copy",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageChannels); return &p }(),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			channel, err := s.State.Channel(i.ChannelID)
			if err != nil {
				channel, err = s.Channel(i.ChannelID)
				if err != nil {
					SendError(s, i, "Failed to fetch channel.")
					return
				}
			}

			// Prepare new channel data
			createData := discordgo.GuildChannelCreateData{
				Name:                 channel.Name,
				Type:                 channel.Type,
				Topic:                channel.Topic,
				Bitrate:              channel.Bitrate,
				UserLimit:            channel.UserLimit,
				RateLimitPerUser:     channel.RateLimitPerUser,
				Position:             channel.Position,
				PermissionOverwrites: channel.PermissionOverwrites,
				ParentID:             channel.ParentID,
				NSFW:                 channel.NSFW,
			}

			// Recreate the channel
			newChannel, err := s.GuildChannelCreateComplex(channel.GuildID, createData)
			if err != nil {
				SendError(s, i, fmt.Sprintf("Failed to clone channel: %v", err))
				return
			}

			// Delete old channel
			_, err = s.ChannelDelete(channel.ID)
			if err != nil {
				// If we fail to delete, try to delete the newly created one to avoid dupes
				_, _ = s.ChannelDelete(newChannel.ID)
				SendError(s, i, fmt.Sprintf("Failed to delete original channel: %v", err))
				return
			}

			// Send success message to new channel
			_, _ = s.ChannelMessageSend(newChannel.ID, "💣 This channel has been nuked.")
		},
	}
}
