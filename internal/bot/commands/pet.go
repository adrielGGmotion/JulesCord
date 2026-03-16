package commands

import (
	"context"
	"errors"
	"fmt"
	"julescord/internal/db"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5"
)

func Pet(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "pet",
			Description: "Adopt and care for your virtual pet",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "adopt",
					Description: "Adopt a new pet",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "type",
							Description: "The type of pet to adopt",
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "Dog 🐶", Value: "Dog"},
								{Name: "Cat 🐱", Value: "Cat"},
								{Name: "Rabbit 🐰", Value: "Rabbit"},
								{Name: "Turtle 🐢", Value: "Turtle"},
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The name for your new pet",
							Required:    true,
						},
					},
				},
				{
					Name:        "view",
					Description: "View your pet's stats",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "feed",
					Description: "Feed your pet",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "play",
					Description: "Play with your pet",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			var user *discordgo.User
			if i.Member != nil {
				user = i.Member.User
			} else {
				user = i.User
			}

			if user == nil {
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "adopt":
				handlePetAdopt(s, i, database, user, subcommand.Options)
			case "view":
				handlePetView(s, i, database, user)
			case "feed":
				handlePetFeed(s, i, database, user)
			case "play":
				handlePetPlay(s, i, database, user)
			}
		},
	}
}

func handlePetAdopt(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, user *discordgo.User, options []*discordgo.ApplicationCommandInteractionDataOption) {
	ctx := context.Background()

	// Check if already has a pet
	existingPet, err := database.GetPet(ctx, i.GuildID, user.ID)
	if err == nil && existingPet != nil {
		SendError(s, i, fmt.Sprintf("You already have a pet named **%s**! You can only have one pet.", existingPet.Name))
		return
	}

	var petType, petName string
	for _, opt := range options {
		if opt.Name == "type" {
			petType = opt.StringValue()
		} else if opt.Name == "name" {
			petName = opt.StringValue()
		}
	}

	if len(petName) > 32 {
		SendError(s, i, "Pet name must be 32 characters or less.")
		return
	}

	err = database.AdoptPet(ctx, i.GuildID, user.ID, petName, petType)
	if err != nil {
		slog.Error("Failed to adopt pet", "guild", i.GuildID, "user", user.ID, "err", err)
		SendError(s, i, "Failed to adopt pet. Please try again later.")
		return
	}

	emoji := "🐾"
	switch petType {
	case "Dog":
		emoji = "🐶"
	case "Cat":
		emoji = "🐱"
	case "Rabbit":
		emoji = "🐰"
	case "Turtle":
		emoji = "🐢"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🎉 Congratulations! You have adopted a %s named **%s** %s", petType, petName, emoji),
		},
	})
}

func handlePetView(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, user *discordgo.User) {
	ctx := context.Background()

	pet, err := database.GetPet(ctx, i.GuildID, user.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || err.Error() == "no rows in result set" {
			SendError(s, i, "You don't have a pet yet! Use `/pet adopt` to get one.")
		} else {
			slog.Error("Failed to fetch pet", "guild", i.GuildID, "user", user.ID, "err", err)
			SendError(s, i, "Failed to fetch your pet.")
		}
		return
	}

	emoji := "🐾"
	switch pet.Type {
	case "Dog":
		emoji = "🐶"
	case "Cat":
		emoji = "🐱"
	case "Rabbit":
		emoji = "🐰"
	case "Turtle":
		emoji = "🐢"
	}

	status := "Happy and well-fed!"
	if pet.Hunger > 80 || pet.Happiness < 20 {
		status = "Your pet needs attention! Use `/pet feed` or `/pet play`."
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s the %s %s", pet.Name, pet.Type, emoji),
		Description: status,
		Color:       0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Hunger",
				Value:  fmt.Sprintf("%d/100", pet.Hunger),
				Inline: true,
			},
			{
				Name:   "Happiness",
				Value:  fmt.Sprintf("%d/100", pet.Happiness),
				Inline: true,
			},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handlePetFeed(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, user *discordgo.User) {
	ctx := context.Background()

	pet, err := database.GetPet(ctx, i.GuildID, user.ID)
	if err != nil {
		SendError(s, i, "You don't have a pet yet! Use `/pet adopt` to get one.")
		return
	}

	if pet.Hunger == 0 {
		SendError(s, i, fmt.Sprintf("**%s** is already full!", pet.Name))
		return
	}

	err = database.FeedPet(ctx, i.GuildID, user.ID)
	if err != nil {
		slog.Error("Failed to feed pet", "err", err)
		SendError(s, i, "Failed to feed your pet.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🍖 You fed **%s**! Their hunger decreased and they are happier.", pet.Name),
		},
	})
}

func handlePetPlay(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, user *discordgo.User) {
	ctx := context.Background()

	pet, err := database.GetPet(ctx, i.GuildID, user.ID)
	if err != nil {
		SendError(s, i, "You don't have a pet yet! Use `/pet adopt` to get one.")
		return
	}

	if pet.Happiness == 100 {
		SendError(s, i, fmt.Sprintf("**%s** is already max happiness!", pet.Name))
		return
	}

	err = database.PlayPet(ctx, i.GuildID, user.ID)
	if err != nil {
		slog.Error("Failed to play with pet", "err", err)
		SendError(s, i, "Failed to play with your pet.")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🎾 You played with **%s**! Their happiness increased (but they are a little hungrier now).", pet.Name),
		},
	})
}
