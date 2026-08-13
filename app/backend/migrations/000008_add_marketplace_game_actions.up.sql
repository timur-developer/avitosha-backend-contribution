ALTER TABLE user_actions DROP CONSTRAINT user_actions_action_type_check;
ALTER TABLE user_actions ADD CONSTRAINT user_actions_action_type_check CHECK (action_type IN ('AD_VIEWED', 'AD_FAVORITED', 'MESSAGE_SENT', 'AD_CREATED', 'DELIVERY_USED', 'REVIEW_LEFT', 'BOOKING_CREATED', 'LISTING_IMPROVED', 'LISTING_SOLD'));

ALTER TABLE pet_activity_scores ADD COLUMN quality_score INTEGER NOT NULL DEFAULT 0 CHECK (quality_score >= 0);

ALTER TABLE domain_events DROP CONSTRAINT domain_events_event_type_check;
ALTER TABLE domain_events ADD CONSTRAINT domain_events_event_type_check CHECK (event_type IN ('TASK_PROGRESS_UPDATED', 'TASK_COMPLETED', 'XP_EARNED', 'PET_LEVEL_UP', 'PET_MOOD_CHANGED', 'ROOM_ITEM_UNLOCKED', 'STORY_STAGE_COMPLETED', 'STORY_COMPLETED', 'LEADERBOARD_SCORE_UPDATED', 'ACHIEVEMENT_UNLOCKED', 'PET_CHARACTER_UNLOCKED', 'AVITO_REWARD_EARNED', 'REWARD_CATALOG_UNLOCKED', 'DAILY_QUEST_UPDATED', 'DAILY_QUEST_COMPLETED', 'STREAK_UPDATED', 'LISTING_VIEWED', 'LISTING_FAVORITED', 'SELLER_CONTACTED', 'LISTING_PUBLISHED', 'LISTING_IMPROVED', 'LISTING_SOLD', 'DELIVERY_USED'));

CREATE TABLE product_action_rules (
    action_type TEXT PRIMARY KEY,
    xp_reward INTEGER NOT NULL CHECK (xp_reward >= 0)
);

INSERT INTO product_action_rules (action_type, xp_reward) VALUES
    ('AD_VIEWED', 0), ('AD_FAVORITED', 5), ('MESSAGE_SENT', 8), ('AD_CREATED', 40),
    ('LISTING_IMPROVED', 20), ('LISTING_SOLD', 50), ('DELIVERY_USED', 15),
    ('REVIEW_LEFT', 0), ('BOOKING_CREATED', 0);

CREATE TABLE marketplace_game_requests (
    event_id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    listing_id UUID NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    operation TEXT NOT NULL,
    result JSONB NOT NULL DEFAULT '{}'::JSONB,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (operation IN ('VIEW', 'FAVORITE', 'MESSAGE', 'PUBLISH', 'IMPROVE', 'PURCHASE')),
    CHECK (jsonb_typeof(result) = 'object')
);

CREATE TABLE listing_quality_awards (
    listing_id UUID NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    criterion TEXT NOT NULL,
    awarded_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (listing_id, criterion),
    CHECK (criterion IN ('price', 'photo', 'description'))
);
