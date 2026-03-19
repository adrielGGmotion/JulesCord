package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5"
)

// parseDuration helper (already exists in some files, but we need one locally or just use time.ParseDuration)
func parseDurationStr(durationStr string) (time.Duration, error) {
	if strings.HasSuffix(durationStr, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(durationStr, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(durationStr)
}

// TempNick returns the /tempnick command.
func TempNick(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "tempnick",
			Description:              "Manage temporary nicknames for users",
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageNicknames),
			DMPermission:             func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Set a temporary nickname for a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The user to set the nickname for",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
						{
							Name:        "nickname",
							Description: "The temporary nickname to set",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "duration",
							Description: "Duration of the nickname (e.g. 1h, 10m, 1d)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a temporary nickname early",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The user to revert the nickname for",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			subcommand := i.ApplicationCommandData().Options[0]
			ctx := context.Background()

			if subcommand.Name == "set" {
				userOpt := subcommand.Options[0].UserValue(s)
				nicknameOpt := subcommand.Options[1].StringValue()
				durationStr := subcommand.Options[2].StringValue()

				duration, err := parseDurationStr(durationStr)
				if err != nil {
					SendError(s, i, "Invalid duration format. Use 1h, 10m, 1d, etc.")
					return
				}

				member, err := s.GuildMember(i.GuildID, userOpt.ID)
				if err != nil {
					SendError(s, i, "Failed to get user information.")
					return
				}

				originalNickname := member.Nick

				err = s.GuildMemberNickname(i.GuildID, userOpt.ID, nicknameOpt)
				if err != nil {
					SendError(s, i, "Failed to set user nickname. Ensure my role is higher than the user's.")
					return
				}

				expiresAt := time.Now().Add(duration)
				err = database.SetTempNickname(ctx, i.GuildID, userOpt.ID, originalNickname, expiresAt)
				if err != nil {
					SendError(s, i, "Failed to save temporary nickname to database.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Title:       "Temporary Nickname Set",
								Description: fmt.Sprintf("Changed <@%s>'s nickname to **%s**.\nExpires: <t:%d:R>", userOpt.ID, nicknameOpt, expiresAt.Unix()),
								Color:       0x00FF00,
							},
						},
					},
				})

			} else if subcommand.Name == "remove" {
				userOpt := subcommand.Options[0].UserValue(s)

				tn, err := database.GetTempNicknameByGuildUser(ctx, i.GuildID, userOpt.ID)
				if err != nil {
					if err == pgx.ErrNoRows {
						SendError(s, i, "This user does not have an active temporary nickname.")
					} else {
						SendError(s, i, "Database error while fetching temporary nickname.")
					}
					return
				}

				err = s.GuildMemberNickname(i.GuildID, userOpt.ID, tn.OriginalNickname)
				if err != nil {
					SendError(s, i, "Failed to revert user nickname. Ensure my role is higher than the user's.")
					return
				}

				err = database.RemoveTempNicknameByGuildUser(ctx, i.GuildID, userOpt.ID)
				if err != nil {
					SendError(s, i, "Failed to remove temporary nickname from database.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Title:       "Temporary Nickname Removed",
								Description: fmt.Sprintf("Reverted <@%s>'s nickname to **%s**.", userOpt.ID, tn.OriginalNickname),
								Color:       0x00FF00,
							},
						},
					},
				})
			}
		},
	}
}
