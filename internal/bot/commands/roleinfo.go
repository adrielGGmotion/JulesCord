package commands

import (
	"fmt"
	"julescord/internal/db"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Roleinfo returns a command that displays detailed information about a role.
func Roleinfo(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "roleinfo",
			Description: "Display detailed information about a role",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "The role to get information about",
					Required:    true,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			if len(i.ApplicationCommandData().Options) == 0 {
				SendError(s, i, "Please provide a role.")
				return
			}

			roleID := i.ApplicationCommandData().Options[0].Value.(string)

			guild, err := s.State.Guild(i.GuildID)
			if err != nil {
				guild, err = s.Guild(i.GuildID)
				if err != nil {
					SendError(s, i, "Failed to retrieve roles for this server.")
					return
				}
			}
			roles := guild.Roles

			var targetRole *discordgo.Role
			for _, r := range roles {
				if r.ID == roleID {
					targetRole = r
					break
				}
			}

			if targetRole == nil {
				SendError(s, i, "Could not find that role.")
				return
			}

			// Format creation date from Snowflake ID
			snowflake, err := discordgo.SnowflakeTimestamp(targetRole.ID)
			var createdAt string
			if err == nil {
				createdAt = fmt.Sprintf("<t:%d:f> (<t:%d:R>)", snowflake.Unix(), snowflake.Unix())
			} else {
				createdAt = "Unknown"
			}

			hoist := "No"
			if targetRole.Hoist {
				hoist = "Yes"
			}

			mentionable := "No"
			if targetRole.Mentionable {
				mentionable = "Yes"
			}

			managed := "No"
			if targetRole.Managed {
				managed = "Yes"
			}

			embed := &discordgo.MessageEmbed{
				Title:       "Role Information",
				Description: fmt.Sprintf("<@&%s>", targetRole.ID),
				Color:       targetRole.Color,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Name",
						Value:  targetRole.Name,
						Inline: true,
					},
					{
						Name:   "Role ID",
						Value:  targetRole.ID,
						Inline: true,
					},
					{
						Name:   "Color (Hex)",
						Value:  fmt.Sprintf("#%06X", targetRole.Color),
						Inline: true,
					},
					{
						Name:   "Position",
						Value:  fmt.Sprintf("%d", targetRole.Position),
						Inline: true,
					},
					{
						Name:   "Hoisted",
						Value:  hoist,
						Inline: true,
					},
					{
						Name:   "Mentionable",
						Value:  mentionable,
						Inline: true,
					},
					{
						Name:   "Managed/Integration",
						Value:  managed,
						Inline: true,
					},
					{
						Name:   "Permissions",
						Value:  fmt.Sprintf("%d", targetRole.Permissions),
						Inline: true,
					},
					{
						Name:   "Created",
						Value:  createdAt,
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
