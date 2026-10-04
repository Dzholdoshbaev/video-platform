CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    age INTEGER NOT NULL CHECK (age >= 13),
    country TEXT NOT NULL,
    premium BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE subscriptions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan TEXT NOT NULL,
    price NUMERIC(12, 2) NOT NULL CHECK (price > 0),
    auto_renew BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX subscriptions_user_id_idx ON subscriptions(user_id);

CREATE TABLE episodes (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    season INTEGER NOT NULL CHECK (season > 0),
    duration INTEGER NOT NULL CHECK (duration > 0)
);

CREATE TABLE watch_history (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    episode_id TEXT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    watched_percentage DOUBLE PRECISION NOT NULL CHECK (watched_percentage BETWEEN 0 AND 100),
    last_position INTEGER NOT NULL CHECK (last_position >= 0),
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (user_id, episode_id)
);
CREATE INDEX watch_history_user_updated_idx ON watch_history(user_id, updated_at DESC);

CREATE TABLE favorites (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_id TEXT NOT NULL,
    PRIMARY KEY (user_id, content_id)
);

CREATE TABLE analytics_events (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN (
        'play_started', 'play_stopped', 'play_completed', 'liked', 'added_to_favorites'
    )),
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX analytics_events_user_created_idx ON analytics_events(user_id, created_at DESC);
CREATE INDEX analytics_events_content_type_idx ON analytics_events(content_id, type);
