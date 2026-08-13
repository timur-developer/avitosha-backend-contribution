CREATE TABLE listing_favorite_rewards (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    listing_id UUID NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    awarded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, listing_id)
);
