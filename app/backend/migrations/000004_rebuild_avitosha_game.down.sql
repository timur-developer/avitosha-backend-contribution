DROP TABLE IF EXISTS domain_events;
DROP TABLE IF EXISTS pet_activity_scores;
DROP TABLE IF EXISTS user_achievements;
DROP TABLE IF EXISTS achievements;
DROP TABLE IF EXISTS daily_progress;
DROP TABLE IF EXISTS weekly_progress;
DROP TABLE IF EXISTS user_room_items;
DROP TABLE IF EXISTS user_actions;
DROP TABLE IF EXISTS user_tasks;
DROP TABLE IF EXISTS user_story_progress;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS pets;
DROP TABLE IF EXISTS stories;
DROP TABLE IF EXISTS room_items;

CREATE TABLE pets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    level SMALLINT NOT NULL DEFAULT 1,
    growth_xp INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id),
    CHECK (btrim(name) <> ''),
    CHECK (level BETWEEN 1 AND 5),
    CHECK (growth_xp >= 0)
);

CREATE TABLE pet_daily_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pet_id UUID NOT NULL REFERENCES pets (id) ON DELETE CASCADE,
    date DATE NOT NULL,
    satiety SMALLINT NOT NULL,
    mood SMALLINT NOT NULL,
    curiosity SMALLINT NOT NULL,
    state TEXT NOT NULL,
    happy_xp_granted BOOLEAN NOT NULL DEFAULT FALSE,
    ecstatic_xp_granted BOOLEAN NOT NULL DEFAULT FALSE,
    starting_growth_xp INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (pet_id, date),
    CHECK (satiety BETWEEN 0 AND 100),
    CHECK (mood BETWEEN 0 AND 100),
    CHECK (curiosity BETWEEN 0 AND 100),
    CHECK (state IN ('CURIOUS', 'HUNGRY', 'BORED', 'CONTENT', 'HAPPY', 'ECSTATIC'))
);

CREATE TABLE inventory_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    item_type TEXT NOT NULL,
    status TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_id UUID NOT NULL,
    idempotency_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at TIMESTAMPTZ,
    UNIQUE (user_id, idempotency_key),
    CHECK (item_type IN ('FOOD', 'TOY', 'BOOK')),
    CHECK (status IN ('AVAILABLE', 'USED', 'EXPIRED'))
);
