package commands

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// NewEmojiManagerCommand creates the /emojimanager command
func NewEmojiManagerCommand() *Command {
	perm := int64(1073741824) // discordgo.PermissionManageEmojis
	dmPerm := false
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "emojimanager",
			Description:              "Manage custom server emojis",
			DefaultMemberPermissions: &perm,
			DMPermission:             &dmPerm,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a new custom emoji",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The name of the new emoji",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "url",
							Description: "The URL of the image",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a custom emoji",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "emoji",
							Description: "The exact name or ID of the emoji",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all custom emojis",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subCommand := options[0]

			switch subCommand.Name {
			case "add":
				handleAddEmoji(s, i, subCommand.Options)
			case "remove":
				handleRemoveEmoji(s, i, subCommand.Options)
			case "list":
				handleListEmojis(s, i)
			}
		},
	}
}

func handleAddEmoji(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		slog.Error("Failed to defer interaction", "error", err)
		return
	}

	if len(options) < 2 {
		msg := "Missing required options."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	var name, url string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.Value.(string)
		} else if opt.Name == "url" {
			url = opt.Value.(string)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		msg := "Failed to fetch the image URL."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Failed to fetch image, status code: %d", resp.StatusCode)
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	// Read up to 5MB
	limitReader := io.LimitReader(resp.Body, 5*1024*1024)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		msg := "Failed to read the image data."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	if len(data) == 0 {
		msg := "Image is empty."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		msg := "The provided URL does not point to a valid image."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	base64Str := fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data))

	emoji, err := s.GuildEmojiCreate(i.GuildID, &discordgo.EmojiParams{
		Name:  name,
		Image: base64Str,
	})
	if err != nil {
		slog.Error("Failed to create emoji", "error", err)
		msg := "Failed to create the emoji. The image may be too large or invalid."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	msg := fmt.Sprintf("Successfully added emoji: <:%s:%s>", emoji.Name, emoji.ID)
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
}

func handleRemoveEmoji(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		slog.Error("Failed to defer interaction", "error", err)
		return
	}

	if len(options) == 0 {
		msg := "Missing emoji parameter."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	target := options[0].Value.(string)

	emojis, err := s.GuildEmojis(i.GuildID)
	if err != nil {
		msg := "Failed to fetch emojis for this server."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	var matchedEmoji *discordgo.Emoji
	for _, e := range emojis {
		if e.ID == target || e.Name == target {
			matchedEmoji = e
			break
		}
	}

	if matchedEmoji == nil {
		msg := fmt.Sprintf("Emoji '%s' not found.", target)
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	err = s.GuildEmojiDelete(i.GuildID, matchedEmoji.ID)
	if err != nil {
		slog.Error("Failed to delete emoji", "error", err)
		msg := fmt.Sprintf("Failed to delete emoji '%s'.", matchedEmoji.Name)
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	msg := fmt.Sprintf("Successfully removed emoji '%s'.", matchedEmoji.Name)
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
}

func handleListEmojis(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		slog.Error("Failed to defer interaction", "error", err)
		return
	}

	emojis, err := s.GuildEmojis(i.GuildID)
	if err != nil {
		slog.Error("Failed to fetch emojis", "error", err)
		msg := "Failed to fetch emojis for this server."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	if len(emojis) == 0 {
		msg := "This server has no custom emojis."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})
		return
	}

	var embeds []*discordgo.MessageEmbed
	var currentDescription strings.Builder
	embedCount := 1

	for _, e := range emojis {
		emojiStr := fmt.Sprintf("<:%s:%s> `%s`\n", e.Name, e.ID, e.Name)
		if e.Animated {
			emojiStr = fmt.Sprintf("<a:%s:%s> `%s`\n", e.Name, e.ID, e.Name)
		}

		if currentDescription.Len()+len(emojiStr) > 4000 {
			embeds = append(embeds, &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("Server Emojis (Page %d)", embedCount),
				Description: currentDescription.String(),
				Color:       0x00FF00,
			})
			currentDescription.Reset()
			embedCount++
		}
		currentDescription.WriteString(emojiStr)
	}

	if currentDescription.Len() > 0 {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Server Emojis (Page %d)", embedCount),
			Description: currentDescription.String(),
			Color:       0x00FF00,
		})
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &embeds})
}
