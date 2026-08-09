DROP TABLE IF EXISTS user_daily_quests;
DROP TABLE IF EXISTS daily_quest_templates;
DROP TABLE IF EXISTS user_streaks;
DROP TABLE IF EXISTS reward_catalog_items;

DROP INDEX IF EXISTS idx_reward_transactions_source;

ALTER TABLE reward_transactions
    DROP CONSTRAINT IF EXISTS reward_transactions_source_kind_check;

ALTER TABLE reward_transactions
    DROP COLUMN IF EXISTS source_title,
    DROP COLUMN IF EXISTS source_ref,
    DROP COLUMN IF EXISTS source_kind;

DELETE FROM reward_transactions
WHERE task_id IS NULL;

ALTER TABLE reward_transactions
    ALTER COLUMN task_id SET NOT NULL;

ALTER TABLE reward_transactions
    ADD CONSTRAINT reward_transactions_action_id_task_id_reward_type_key
    UNIQUE (action_id, task_id, reward_type);

ALTER TABLE domain_events
    DROP CONSTRAINT IF EXISTS domain_events_event_type_check;

ALTER TABLE domain_events
    ADD CONSTRAINT domain_events_event_type_check
    CHECK (
        event_type IN (
            'TASK_PROGRESS_UPDATED',
            'TASK_COMPLETED',
            'XP_EARNED',
            'PET_LEVEL_UP',
            'PET_MOOD_CHANGED',
            'ROOM_ITEM_UNLOCKED',
            'STORY_STAGE_COMPLETED',
            'STORY_COMPLETED',
            'LEADERBOARD_SCORE_UPDATED',
            'ACHIEVEMENT_UNLOCKED',
            'PET_CHARACTER_UNLOCKED',
            'AVITO_REWARD_EARNED'
        )
    );
