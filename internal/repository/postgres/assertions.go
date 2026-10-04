package postgres

import "streamingplatform/internal/repository"

// Compile-time checks document implicit interface satisfaction without Java-style declarations.
var (
	_ repository.UserRepository         = (*UserRepository)(nil)
	_ repository.SubscriptionRepository = (*SubscriptionRepository)(nil)
	_ repository.WatchHistoryRepository = (*WatchHistoryRepository)(nil)
	_ repository.FavoriteRepository     = (*FavoriteRepository)(nil)
)
