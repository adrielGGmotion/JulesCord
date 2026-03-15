package commands

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/metrics"
)

// Command defines the structure for a slash command.
type Command struct {
	Definition *discordgo.ApplicationCommand
	Handler    func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// Registry manages the registration and dispatching of commands.
type Registry struct {
	Commands map[string]*Command
}

// NewRegistry creates a new command registry.
func NewRegistry() *Registry {
	return &Registry{
		Commands: make(map[string]*Command),
	}
}

// Add registers a command in the registry.
func (r *Registry) Add(cmd *Command) {
	if cmd == nil || cmd.Definition == nil {
		slog.Info("Attempted to register nil command or nil definition")
		return
	}
	r.Commands[cmd.Definition.Name] = cmd
}

// Dispatch routes the interaction to the appropriate command handler.
func (r *Registry) Dispatch(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	name := i.ApplicationCommandData().Name
	if cmd, ok := r.Commands[name]; ok {
		start := time.Now()
		metrics.CommandCounter.WithLabelValues(name).Inc()
		defer func() {
			metrics.CommandLatency.WithLabelValues(name).Observe(time.Since(start).Seconds())
		}()

		cmd.Handler(s, i)
	} else {
		metrics.ErrorCounter.WithLabelValues("unknown_command").Inc()
		slog.Info(fmt.Sprintf("Received interaction for unknown command: %s", name))
	}
}

// RegisterWithDiscord registers all commands in the registry with Discord.
func (r *Registry) RegisterWithDiscord(s *discordgo.Session, appID string, guildID string) error {
	slog.Info("Registering slash commands...")

	// Create a slice of application commands to register all at once
	var commands []*discordgo.ApplicationCommand
	for _, cmd := range r.Commands {
		commands = append(commands, cmd.Definition)
	}

	if appID == "" {
		appID = s.State.User.ID
	}

	_, err := s.ApplicationCommandBulkOverwrite(appID, guildID, commands)
	if err != nil {
		metrics.ErrorCounter.WithLabelValues("command_registration").Inc()
		slog.Error("Cannot overwrite commands", "error", err)
		return err
	}

	slog.Info(fmt.Sprintf("Slash commands registered successfully. Count: %d", len(commands)))
	return nil
}
