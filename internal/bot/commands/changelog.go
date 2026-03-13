package commands

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

type GitHubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
	HTMLURL string `json:"html_url"`
}

func Changelog() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "changelog",
			Description: "Displays the 5 most recent commits to the JulesCord repository",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Acknowledge the interaction immediately since API call might take time
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				slog.Error("Failed to defer interaction for changelog", "error", err)
				return
			}

			client := &http.Client{Timeout: 5 * time.Second}
			req, err := http.NewRequest("GET", "https://api.github.com/repos/adrielGGmotion/JulesCord/commits?per_page=5", nil)
			if err != nil {
				sendErrorResponse(s, i.Interaction, "Failed to create request to GitHub API.")
				return
			}

			// Add a User-Agent header as requested by GitHub API
			req.Header.Set("User-Agent", "JulesCord-Bot")

			resp, err := client.Do(req)
			if err != nil {
				sendErrorResponse(s, i.Interaction, "Failed to fetch commits from GitHub.")
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				sendErrorResponse(s, i.Interaction, fmt.Sprintf("GitHub API returned status: %s", resp.Status))
				return
			}

			var commits []GitHubCommit
			if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
				sendErrorResponse(s, i.Interaction, "Failed to parse commits from GitHub.")
				return
			}

			if len(commits) == 0 {
				sendErrorResponse(s, i.Interaction, "No commits found in the repository.")
				return
			}

			var description string
			for _, c := range commits {
				// Get first line of commit message
				msg := c.Commit.Message
				if len(msg) > 100 {
					msg = msg[:97] + "..."
				}

				// Format SHA to 7 chars
				shortSHA := c.SHA
				if len(shortSHA) > 7 {
					shortSHA = shortSHA[:7]
				}

				description += fmt.Sprintf("[`%s`](%s) **%s** - %s\n\n", shortSHA, c.HTMLURL, msg, c.Commit.Author.Name)
			}

			embed := &discordgo.MessageEmbed{
				Title:       "JulesCord Changelog (Recent Commits)",
				Description: description,
				Color:       0x00ff00,
				URL:         "https://github.com/adrielGGmotion/JulesCord/commits/main",
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Fetched directly from GitHub API",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
			if err != nil {
				slog.Error("Failed to edit interaction response with changelog", "error", err)
			}
		},
	}
}

func sendErrorResponse(s *discordgo.Session, i *discordgo.Interaction, message string) {
	_, err := s.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content: &message,
	})
	if err != nil {
		slog.Error("Failed to send error response for changelog", "error", err)
	}
}
