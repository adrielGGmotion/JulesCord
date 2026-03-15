package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

func stringPtr(s string) *string {
	return &s
}

// OpenTDBResponse represents the response from the OpenTDB API.
type OpenTDBResponse struct {
	ResponseCode int `json:"response_code"`
	Results      []struct {
		Category         string   `json:"category"`
		Type             string   `json:"type"`
		Difficulty       string   `json:"difficulty"`
		Question         string   `json:"question"`
		CorrectAnswer    string   `json:"correct_answer"`
		IncorrectAnswers []string `json:"incorrect_answers"`
	} `json:"results"`
}

// NewTriviaCommand returns the trivia command definition and handler.
func NewTriviaCommand(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "trivia",
			Description: "Play trivia to earn coins!",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Start a new trivia question",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "leaderboard",
					Description: "View the trivia leaderboard",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil || i.Member.User == nil {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "This command can only be used in a server.",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			switch options[0].Name {
			case "start":
				handleTriviaStart(s, i)
			case "leaderboard":
				handleTriviaLeaderboard(s, i, database)
			}
		},
	}
}

func handleTriviaStart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Acknowledge interaction first as fetching might take time
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		slog.Error("Failed to acknowledge trivia start", "error", err)
		return
	}

	resp, err := http.Get("https://opentdb.com/api.php?amount=1")
	if err != nil {
		slog.Error("Failed to fetch trivia question", "error", err)
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("Failed to fetch trivia question. Please try again later."),
		})
		return
	}
	defer resp.Body.Close()

	var tdbResp OpenTDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&tdbResp); err != nil {
		slog.Error("Failed to decode trivia question", "error", err)
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("Failed to decode trivia question. Please try again later."),
		})
		return
	}

	if tdbResp.ResponseCode != 0 || len(tdbResp.Results) == 0 {
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("Failed to find a trivia question."),
		})
		return
	}

	questionData := tdbResp.Results[0]
	questionText := html.UnescapeString(questionData.Question)
	correctAnswer := html.UnescapeString(questionData.CorrectAnswer)

	answers := []string{correctAnswer}
	for _, a := range questionData.IncorrectAnswers {
		answers = append(answers, html.UnescapeString(a))
	}

	// Shuffle answers
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(answers), func(i, j int) {
		answers[i], answers[j] = answers[j], answers[i]
	})

	correctIndex := -1
	for idx, a := range answers {
		if a == correctAnswer {
			correctIndex = idx
			break
		}
	}

	// Use custom ID to store correct index: trivia_{correctIndex}_{timestampNano}
	// To prevent button brute forcing, the timestamp ensures uniqueness and we can track it or just let the first click win
	// In interaction handler, we will respond with UpdateMessage to disable buttons.
	uniqueID := time.Now().UnixNano()

	var components []discordgo.MessageComponent
	var actionRowComponents []discordgo.MessageComponent

	for idx, a := range answers {
		// customID format: trivia_answer_{is_correct}_{uniqueID}_{index}
		// is_correct is "1" or "0"
		isCorrectStr := "0"
		if idx == correctIndex {
			isCorrectStr = "1"
		}

		customID := fmt.Sprintf("trivia_answer_%s_%d_%d", isCorrectStr, uniqueID, idx)

		// Truncate answer if too long for button label
		label := a
		if len(label) > 80 {
			label = string([]rune(label)[:77]) + "..."
		}

		actionRowComponents = append(actionRowComponents, discordgo.Button{
			Label:    label,
			Style:    discordgo.PrimaryButton,
			CustomID: customID,
		})
	}

	components = append(components, discordgo.ActionsRow{
		Components: actionRowComponents,
	})

	embed := &discordgo.MessageEmbed{
		Title:       "Trivia Question",
		Description: fmt.Sprintf("**Category:** %s\n**Difficulty:** %s\n\n%s", html.UnescapeString(questionData.Category), questionData.Difficulty, questionText),
		Color:       0x3498db,
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		slog.Error("Failed to edit interaction response with trivia question", "error", err)
	}
}

func handleTriviaLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB) {
	if database == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Database is not connected.",
			},
		})
		return
	}

	leaders, err := database.GetTriviaLeaderboard(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to get trivia leaderboard", "error", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to fetch leaderboard.",
			},
		})
		return
	}

	if len(leaders) == 0 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No trivia scores found in this server.",
			},
		})
		return
	}

	desc := ""
	for idx, leader := range leaders {
		desc += fmt.Sprintf("**%d.** <@%s> - %d points\n", idx+1, leader.UserID, leader.Score)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏆 Trivia Leaderboard",
		Description: desc,
		Color:       0xf1c40f,
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
