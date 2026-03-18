package commands

import (
	"fmt"
	"julescord/internal/db"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Userinfo returns a command that displays detailed information about a user.
func Userinfo(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "userinfo",
			Description: "Display detailed information about a user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to get information about (defaults to yourself)",
					Required:    false,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			var targetMember *discordgo.Member
			var targetUser *discordgo.User

			if len(i.ApplicationCommandData().Options) > 0 {
				opt := i.ApplicationCommandData().Options[0]
				targetUser = opt.UserValue(s)
				if targetUser != nil {
					var err error
					targetMember, err = s.State.Member(i.GuildID, targetUser.ID)
					if err != nil {
						targetMember, err = s.GuildMember(i.GuildID, targetUser.ID)
						if err != nil {
							targetMember = &discordgo.Member{
								User:  targetUser,
								Roles: []string{},
							}
						}
					}
				}
			} else {
				targetMember = i.Member
				targetUser = i.Member.User
			}

			if targetUser == nil {
				SendError(s, i, "Could not find that user.")
				return
			}

			// Extract creation date from Snowflake ID
			snowflake, err := discordgo.SnowflakeTimestamp(targetUser.ID)
			var createdAt string
			if err == nil {
				createdAt = fmt.Sprintf("<t:%d:f> (<t:%d:R>)", snowflake.Unix(), snowflake.Unix())
			} else {
				createdAt = "Unknown"
			}

			// Format join date
			var joinedAt string
			if !targetMember.JoinedAt.IsZero() {
				joinedAt = fmt.Sprintf("<t:%d:f> (<t:%d:R>)", targetMember.JoinedAt.Unix(), targetMember.JoinedAt.Unix())
			} else {
				joinedAt = "Unknown"
			}

			// Format roles
			rolesStr := "None"
			highestRoleStr := "None"
			if len(targetMember.Roles) > 0 {
				var roles []string
				highestPosition := -1

				guild, err := s.State.Guild(i.GuildID)
				if err == nil {
					// Map role ID to Position
					rolePosMap := make(map[string]int)
					for _, r := range guild.Roles {
						rolePosMap[r.ID] = r.Position
					}

					for _, roleID := range targetMember.Roles {
						roles = append(roles, fmt.Sprintf("<@&%s>", roleID))

						if pos, ok := rolePosMap[roleID]; ok {
							if pos > highestPosition {
								highestPosition = pos
								highestRoleStr = fmt.Sprintf("<@&%s>", roleID)
							}
						}
					}
				} else {
					for _, roleID := range targetMember.Roles {
						roles = append(roles, fmt.Sprintf("<@&%s>", roleID))
					}
				}

				rolesStr = strings.Join(roles, ", ")
				// Truncate if too long to avoid embed field limits
				if len(rolesStr) > 1024 {
					rolesStr = rolesStr[:1021] + "..."
				}
			}

			botStatus := "No"
			if targetUser.Bot {
				botStatus = "Yes"
			}

			embed := &discordgo.MessageEmbed{
				Title:       "User Information",
				Description: fmt.Sprintf("<@%s>", targetUser.ID),
				Color:       0x0099ff,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targetUser.AvatarURL(""),
				},
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Username",
						Value:  targetUser.Username,
						Inline: true,
					},
					{
						Name:   "User ID",
						Value:  targetUser.ID,
						Inline: true,
					},
					{
						Name:   "Bot",
						Value:  botStatus,
						Inline: true,
					},
					{
						Name:   "Account Created",
						Value:  createdAt,
						Inline: false,
					},
					{
						Name:   "Joined Server",
						Value:  joinedAt,
						Inline: false,
					},
					{
						Name:   "Highest Role",
						Value:  highestRoleStr,
						Inline: true,
					},
					{
						Name:   fmt.Sprintf("Roles [%d]", len(targetMember.Roles)),
						Value:  rolesStr,
						Inline: false,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Requested by %s", i.Member.User.Username),
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			SendEmbed(s, i, embed)
		},
	}
}
