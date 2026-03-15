package commands

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Rank returns the /rank command definition and handler.
func Rank(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "rank",
			Description: "Displays and manages user ranks and economy features.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "view",
					Description: "Displays your or another user's XP, level, and server rank.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to check the rank of",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "set-background",
					Description: "Set a custom profile background image URL.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "url",
							Description: "The URL of the background image",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "role-rewards",
					Description: "View all level roles in the server.",
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			if database == nil {
				SendError(s, i, "Database is not connected. Cannot fetch rank data.")
				return
			}

			subcommand := i.ApplicationCommandData().Options[0]
			switch subcommand.Name {
			case "view":
				handleRankView(s, i, database, subcommand.Options)
			case "set-background":
				handleRankSetBackground(s, i, database, subcommand.Options)
			case "role-rewards":
				handleRankRoleRewards(s, i, database)
			}
		},
	}
}

func handleRankView(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// Determine target user
	var targetUser *discordgo.User
	for _, option := range options {
		if option.Name == "user" {
			targetUser = option.UserValue(s)
		}
	}

	if targetUser == nil {
		if i.Member != nil {
			targetUser = i.Member.User
		} else {
			targetUser = i.User
		}
	}

	if targetUser.Bot {
		SendError(s, i, "Bots don't earn XP!")
		return
	}

	ctx := context.Background()

	// Fetch economy data
	econ, err := database.GetUserEconomy(ctx, i.GuildID, targetUser.ID)
	if err != nil || econ == nil {
		SendError(s, i, "User has no XP yet.")
		return
	}

	// Fetch rank
	rank, err := database.GetRank(ctx, i.GuildID, targetUser.ID)
	if err != nil {
		slog.Error("Failed to get rank for user %s", "arg1", targetUser.ID, "error", err)
		rank = 0 // Default to 0 if error
	}

	// Send embed
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s's Rank", targetUser.Username),
		Color: 0x3498db, // Blue
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: targetUser.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Rank",
				Value:  fmt.Sprintf("#%d", rank),
				Inline: true,
			},
			{
				Name:   "Level",
				Value:  fmt.Sprintf("%d", econ.Level),
				Inline: true,
			},
			{
				Name:   "XP",
				Value:  fmt.Sprintf("%d", econ.XP),
				Inline: true,
			},
		},
	}

	if econ.BackgroundURL != nil && *econ.BackgroundURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: *econ.BackgroundURL,
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleRankSetBackground(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var url string
	for _, opt := range options {
		if opt.Name == "url" {
			url = opt.StringValue()
		}
	}

	userID := i.Member.User.ID

	err := database.SetBackgroundURL(context.Background(), i.GuildID, userID, url)
	if err != nil {
		slog.Error("Failed to set background url", "error", err)
		SendError(s, i, "Failed to set your background. Make sure you provide a valid image URL.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Your rank profile background has been updated!",
		},
	})
}

func handleRankRoleRewards(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	roles, err := database.GetLevelRoles(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to get level roles", "error", err)
		SendError(s, i, "Failed to retrieve level roles.")
		return
	}

	if len(roles) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "There are no level roles configured for this server.",
			},
		})
		return
	}

	// Sort roles by level ascending
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Level < roles[j].Level
	})

	description := ""
	for _, r := range roles {
		description += fmt.Sprintf("**Level %d:** <@&%s>\n", r.Level, r.RoleID)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Level Role Rewards",
		Description: description,
		Color:       0x9b59b6, // Purple
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
