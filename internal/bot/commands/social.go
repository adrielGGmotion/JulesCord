package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Social creates the /social command
func Social(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "social",
			Description: "Manage social connections",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "follow",
					Description: "Follow a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to follow",
							Required:    true,
						},
					},
				},
				{
					Name:        "unfollow",
					Description: "Unfollow a user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to unfollow",
							Required:    true,
						},
					},
				},
				{
					Name:        "followers",
					Description: "List your followers (or another user's followers)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to check",
							Required:    false,
						},
					},
				},
				{
					Name:        "following",
					Description: "List users you are following (or another user is following)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to check",
							Required:    false,
						},
					},
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

			subcommand := options[0]

			switch subcommand.Name {
			case "follow":
				handleSocialFollow(s, i, database, subcommand.Options)
			case "unfollow":
				handleSocialUnfollow(s, i, database, subcommand.Options)
			case "followers":
				handleSocialFollowers(s, i, database, subcommand.Options)
			case "following":
				handleSocialFollowing(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleSocialFollow(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var targetUser *discordgo.User
	for _, opt := range options {
		if opt.Name == "user" {
			targetUser = opt.UserValue(s)
		}
	}

	if targetUser == nil {
		SendError(s, i, "Invalid user.")
		return
	}

	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	if userID == targetUser.ID {
		SendError(s, i, "You cannot follow yourself.")
		return
	}

	err := database.FollowUser(context.Background(), userID, targetUser.ID)
	if err != nil {
		slog.Error("Failed to follow user", "follower", userID, "following", targetUser.ID, "err", err)
		SendError(s, i, "An error occurred while following the user.")
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "User Followed",
		Description: fmt.Sprintf("You are now following <@%s>.", targetUser.ID),
		Color:       0x00FF00,
	})
}

func handleSocialUnfollow(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var targetUser *discordgo.User
	for _, opt := range options {
		if opt.Name == "user" {
			targetUser = opt.UserValue(s)
		}
	}

	if targetUser == nil {
		SendError(s, i, "Invalid user.")
		return
	}

	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	err := database.UnfollowUser(context.Background(), userID, targetUser.ID)
	if err != nil {
		slog.Error("Failed to unfollow user", "follower", userID, "following", targetUser.ID, "err", err)
		SendError(s, i, "An error occurred while unfollowing the user.")
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "User Unfollowed",
		Description: fmt.Sprintf("You have unfollowed <@%s>.", targetUser.ID),
		Color:       0x00FF00,
	})
}

func handleSocialFollowers(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var targetUser *discordgo.User
	if i.Member != nil && i.Member.User != nil {
		targetUser = i.Member.User
	} else if i.User != nil {
		targetUser = i.User
	}
	for _, opt := range options {
		if opt.Name == "user" {
			targetUser = opt.UserValue(s)
		}
	}

	followers, err := database.GetFollowers(context.Background(), targetUser.ID)
	if err != nil {
		slog.Error("Failed to get followers", "user", targetUser.ID, "err", err)
		SendError(s, i, "Failed to retrieve followers.")
		return
	}

	if len(followers) == 0 {
		SendEmbed(s, i, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s's Followers", targetUser.Username),
			Description: "This user has no followers yet.",
			Color:       0x5865F2,
		})
		return
	}

	var desc strings.Builder
	for _, f := range followers {
		desc.WriteString(fmt.Sprintf("• <@%s>\n", f))
	}

	// Truncate if too long
	if len(desc.String()) > 4000 {
		descStr := desc.String()
		SendEmbed(s, i, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s's Followers (%d)", targetUser.Username, len(followers)),
			Description: descStr[:3990] + "...",
			Color:       0x5865F2,
		})
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s's Followers (%d)", targetUser.Username, len(followers)),
		Description: desc.String(),
		Color:       0x5865F2,
	})
}

func handleSocialFollowing(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var targetUser *discordgo.User
	if i.Member != nil && i.Member.User != nil {
		targetUser = i.Member.User
	} else if i.User != nil {
		targetUser = i.User
	}
	for _, opt := range options {
		if opt.Name == "user" {
			targetUser = opt.UserValue(s)
		}
	}

	following, err := database.GetFollowing(context.Background(), targetUser.ID)
	if err != nil {
		slog.Error("Failed to get following", "user", targetUser.ID, "err", err)
		SendError(s, i, "Failed to retrieve following list.")
		return
	}

	if len(following) == 0 {
		SendEmbed(s, i, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s is Following", targetUser.Username),
			Description: "This user is not following anyone.",
			Color:       0x5865F2,
		})
		return
	}

	var desc strings.Builder
	for _, f := range following {
		desc.WriteString(fmt.Sprintf("• <@%s>\n", f))
	}

	// Truncate if too long
	if len(desc.String()) > 4000 {
		descStr := desc.String()
		SendEmbed(s, i, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s is Following (%d)", targetUser.Username, len(following)),
			Description: descStr[:3990] + "...",
			Color:       0x5865F2,
		})
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s is Following (%d)", targetUser.Username, len(following)),
		Description: desc.String(),
		Color:       0x5865F2,
	})
}
