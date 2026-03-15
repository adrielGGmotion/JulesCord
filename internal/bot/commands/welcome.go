package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Welcome returns the /welcome command.
func Welcome(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "welcome",
			Description: "Configure and test welcome images for your server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup-image",
					Description: "Set an image URL to be included in welcome messages",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "url",
							Description: "The URL of the image",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "test",
					Description: "Test the current welcome message and image configuration",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			// Check for Admin permissions
			if i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
				SendError(s, i, "You need Administrator permissions to configure the welcome system.")
				return
			}

			options := i.ApplicationCommandData().Options
			subcommand := options[0].Name

			switch subcommand {
			case "setup-image":
				handleWelcomeSetupImage(s, i, database, options[0].Options)
			case "test":
				handleWelcomeTest(s, i, database)
			}
		},
	}
}

func handleWelcomeSetupImage(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	url := options[0].StringValue()

	err := database.SetWelcomeImage(context.Background(), i.GuildID, url)
	if err != nil {
		slog.Error("Failed to set welcome image", "guild", i.GuildID, "error", err)
		SendError(s, i, "Failed to set welcome image. Please try again.")
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Welcome Image Configured",
		Description: "The welcome image has been successfully set. You can use `/welcome test` to preview it.",
		Color:       0x00FF00,
		Image: &discordgo.MessageEmbedImage{
			URL: url,
		},
	})
}

func handleWelcomeTest(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	config, err := database.GetGuildConfig(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to get guild config for welcome test", "error", err)
		SendError(s, i, "Failed to load welcome configuration.")
		return
	}

	imageURL, err := database.GetWelcomeImage(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to get welcome image for test", "error", err)
		SendError(s, i, "Failed to load welcome image configuration.")
		return
	}

	if (config.WelcomeChannelID == nil || *config.WelcomeChannelID == "") && imageURL == "" {
		SendError(s, i, "No welcome channel or image is configured. Use `/config set-welcome-channel` and `/welcome setup-image` first.")
		return
	}

	welcomeMsg := fmt.Sprintf("Welcome to the server, <@%s>! We are glad to have you here.", i.Member.User.ID)

	embed := &discordgo.MessageEmbed{
		Description: welcomeMsg,
		Color:       0x00FF00,
	}

	if imageURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: imageURL,
		}
	}

	SendEmbed(s, i, embed)
}
