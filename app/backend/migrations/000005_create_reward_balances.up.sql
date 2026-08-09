CREATE TABLE user_reward_balances (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    reward_type TEXT NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    earned_total BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, reward_type),
    CHECK (btrim(reward_type) <> ''),
    CHECK (balance >= 0),
    CHECK (earned_total >= balance),
    CHECK (updated_at >= created_at)
);

CREATE TABLE reward_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    action_id UUID NOT NULL REFERENCES user_actions (id) ON DELETE RESTRICT,
    task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE RESTRICT,
    reward_type TEXT NOT NULL,
    amount INTEGER NOT NULL,
    balance_after BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (action_id, task_id, reward_type),
    CHECK (btrim(reward_type) <> ''),
    CHECK (amount > 0),
    CHECK (balance_after >= amount)
);

CREATE INDEX idx_reward_transactions_user_time
    ON reward_transactions (user_id, created_at DESC);
