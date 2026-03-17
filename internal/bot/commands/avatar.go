package commands

import (
	"fmt"
	"julescord/internal/db"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Avatar returns a command that displays a user's avatar.
func Avatar(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "avatar",
			Description: "View a user's avatar in high resolution",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user whose avatar you want to view (defaults to yourself)",
					Required:    false,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var targetUser *discordgo.User

			if len(i.ApplicationCommandData().Options) > 0 {
				opt := i.ApplicationCommandData().Options[0]
				targetUser = opt.UserValue(s)
			} else {
				if i.Member != nil {
					targetUser = i.Member.User
				} else if i.User != nil {
					targetUser = i.User
				}
			}

			if targetUser == nil {
				SendError(s, i, "Could not find that user.")
				return
			}

			avatarURL := targetUser.AvatarURL("1024")

			embed := &discordgo.MessageEmbed{
				Title: fmt.Sprintf("%s's Avatar", targetUser.Username),
				Color: 0x0099ff,
				Image: &discordgo.MessageEmbedImage{
					URL: avatarURL,
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Requested by %s", func() string {
						if i.Member != nil {
							return i.Member.User.Username
						}
						return i.User.Username
					}()),
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			SendEmbed(s, i, embed)
		},
	}
}
