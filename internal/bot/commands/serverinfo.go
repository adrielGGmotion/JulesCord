package commands

import (
	"fmt"
	"julescord/internal/db"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Serverinfo returns a command that displays detailed information about the server.
func Serverinfo(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "serverinfo",
			Description: "Display detailed information about the current server",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			// First try to get from state cache
			guild, err := s.State.Guild(i.GuildID)
			if err != nil {
				// Fallback to API if not in state
				guild, err = s.Guild(i.GuildID)
				if err != nil {
					SendError(s, i, "Could not retrieve server information.")
					return
				}
			}

			// Try to fetch the owner user object
			ownerMention := "Unknown"
			ownerUser, err := s.User(guild.OwnerID)
			if err == nil {
				ownerMention = fmt.Sprintf("<@%s>", ownerUser.ID)
			} else {
				ownerMention = fmt.Sprintf("<@%s>", guild.OwnerID)
			}

			// Extract creation date from Snowflake ID
			snowflake, err := discordgo.SnowflakeTimestamp(guild.ID)
			var createdAt string
			if err == nil {
				createdAt = fmt.Sprintf("<t:%d:f> (<t:%d:R>)", snowflake.Unix(), snowflake.Unix())
			} else {
				createdAt = "Unknown"
			}

			// Handle potentially missing values in state
			memberCount := guild.MemberCount
			if memberCount == 0 {
				// MemberCount is sometimes 0 in partial guild objects
				memberCount = len(guild.Members)
			}

			roleCount := len(guild.Roles)
			emojiCount := len(guild.Emojis)
			channelCount := len(guild.Channels)

			// Safely get icon URL
			iconURL := ""
			if guild.Icon != "" {
				iconURL = guild.IconURL("256")
			}

			embed := &discordgo.MessageEmbed{
				Title:       "Server Information",
				Description: fmt.Sprintf("**%s**", guild.Name),
				Color:       0x0099ff,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Owner",
						Value:  ownerMention,
						Inline: true,
					},
					{
						Name:   "Server ID",
						Value:  guild.ID,
						Inline: true,
					},
					{
						Name:   "Created At",
						Value:  createdAt,
						Inline: false,
					},
					{
						Name:   "Members",
						Value:  fmt.Sprintf("%d", memberCount),
						Inline: true,
					},
					{
						Name:   "Channels",
						Value:  fmt.Sprintf("%d", channelCount),
						Inline: true,
					},
					{
						Name:   "Roles",
						Value:  fmt.Sprintf("%d", roleCount),
						Inline: true,
					},
					{
						Name:   "Emojis",
						Value:  fmt.Sprintf("%d", emojiCount),
						Inline: true,
					},
					{
						Name:   "Boost Level",
						Value:  fmt.Sprintf("Level %d", guild.PremiumTier),
						Inline: true,
					},
					{
						Name:   "Boosts",
						Value:  fmt.Sprintf("%d", guild.PremiumSubscriptionCount),
						Inline: true,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Requested by %s", i.Member.User.Username),
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			if iconURL != "" {
				embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
					URL: iconURL,
				}
			}

			SendEmbed(s, i, embed)
		},
	}
}
