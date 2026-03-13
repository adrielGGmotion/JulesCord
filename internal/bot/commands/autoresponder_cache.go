package commands

import (
	"julescord/internal/db"
	"log/slog"
)

// AutoResponderCacheUpdater is an interface that the Bot struct implements
type AutoResponderCacheUpdater interface {
	UpdateAutoResponderCache(guildID string)
}

func updateCache(botInstance interface{}, guildID string, database *db.DB) {
	if updater, ok := botInstance.(AutoResponderCacheUpdater); ok {
		updater.UpdateAutoResponderCache(guildID)
	} else {
		slog.Error("botInstance does not implement AutoResponderCacheUpdater")
	}
}
