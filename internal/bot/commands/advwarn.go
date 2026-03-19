package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
	"julescord/internal/utils"
)

// AdvWarn creates the /advwarn command structure.
func AdvWarn(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "advwarn",
			Description:              "Advanced warning system with expirations",
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageMessages),
			DMPermission:             func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "issue",
					Description: "Issue a new advanced warning",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to warn",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "reason",
							Description: "The reason for the warning",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "duration",
							Description: "Duration until expiration (e.g. 1h, 7d). Leave blank for permanent.",
							Required:    false,
						},
					},
				},
				{
					Name:        "list",
					Description: "List active advanced warnings for a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to check",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a specific advanced warning by ID",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "The ID of the warning to remove",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			subCommand := i.ApplicationCommandData().Options[0].Name
			switch subCommand {
			case "issue":
				handleAdvWarnIssue(s, i, database)
			case "list":
				handleAdvWarnList(s, i, database)
			case "remove":
				handleAdvWarnRemove(s, i, database)
			}
		},
	}
}

func handleAdvWarnIssue(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	options := i.ApplicationCommandData().Options[0].Options
	targetUser := options[0].UserValue(s)
	reason := options[1].StringValue()

	var durationStr string
	var expiresAt *time.Time
	for _, opt := range options {
		if opt.Name == "duration" {
			durationStr = opt.StringValue()
			break
		}
	}

	if durationStr != "" {
		duration, err := utils.ParseDuration(durationStr)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Invalid duration format. Use things like `1h`, `7d`, `30m`.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		t := time.Now().Add(duration)
		expiresAt = &t
	}

	err := database.AddAdvancedWarning(context.Background(), i.GuildID, targetUser.ID, i.Member.User.ID, reason, expiresAt)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to issue advanced warning.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Try to DM the user
	dmChannel, err := s.UserChannelCreate(targetUser.ID)
	if err == nil {
		embed := &discordgo.MessageEmbed{
			Title:       "You have been warned",
			Description: fmt.Sprintf("**Server:** %s\n**Reason:** %s", i.GuildID, reason),
			Color:       0xff0000,
		}
		if expiresAt != nil {
			embed.Description += fmt.Sprintf("\n**Expires:** <t:%d:R>", expiresAt.Unix())
		}
		s.ChannelMessageSendEmbed(dmChannel.ID, embed)
	}

	replyMsg := fmt.Sprintf("Successfully issued advanced warning to <@%s> for `%s`.", targetUser.ID, reason)
	if expiresAt != nil {
		replyMsg += fmt.Sprintf(" Expires <t:%d:R>.", expiresAt.Unix())
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: replyMsg,
		},
	})
}

func handleAdvWarnList(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	options := i.ApplicationCommandData().Options[0].Options
	targetUser := options[0].UserValue(s)

	warnings, err := database.GetAdvancedWarnings(context.Background(), i.GuildID, targetUser.ID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to retrieve warnings.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(warnings) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<@%s> has no active advanced warnings.", targetUser.ID),
			},
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("Active Advanced Warnings for %s", targetUser.Username),
		Color: 0xffa500,
	}

	for _, w := range warnings {
		expiresStr := "Permanent"
		if w.ExpiresAt != nil {
			expiresStr = fmt.Sprintf("<t:%d:R>", w.ExpiresAt.Unix())
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("ID: %d", w.ID),
			Value: fmt.Sprintf("**Reason:** %s\n**Moderator:** <@%s>\n**Issued:** <t:%d:R>\n**Expires:** %s", w.Reason, w.ModeratorID, w.CreatedAt.Unix(), expiresStr),
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleAdvWarnRemove(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	options := i.ApplicationCommandData().Options[0].Options
	warningID := int(options[0].IntValue())

	err := database.RemoveAdvancedWarning(context.Background(), warningID, i.GuildID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to remove warning or it does not exist.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully removed advanced warning ID %d.", warningID),
		},
	})
}
