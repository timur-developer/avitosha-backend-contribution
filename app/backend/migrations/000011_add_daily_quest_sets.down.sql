DROP TABLE IF EXISTS user_daily_goals;

ALTER TABLE user_streaks DROP CONSTRAINT IF EXISTS user_streaks_protection_count_check;
ALTER TABLE user_streaks DROP COLUMN IF EXISTS protection_count;

DELETE FROM user_daily_quests WHERE template_code IN (
    'DAILY_COMPARE', 'DAILY_DIALOG', 'DAILY_NEW_LISTING', 'DAILY_IMPROVE_LISTING'
);

DELETE FROM daily_quest_templates WHERE code IN (
    'DAILY_COMPARE', 'DAILY_DIALOG', 'DAILY_NEW_LISTING', 'DAILY_IMPROVE_LISTING'
);

DELETE FROM user_daily_quests a USING user_daily_quests b
WHERE a.user_id = b.user_id AND a.quest_date = b.quest_date AND a.id > b.id;

ALTER TABLE user_daily_quests DROP CONSTRAINT IF EXISTS user_daily_quests_user_date_template_key;
ALTER TABLE user_daily_quests ADD CONSTRAINT user_daily_quests_user_id_quest_date_key UNIQUE (user_id, quest_date);

ALTER TABLE daily_quest_templates
    DROP CONSTRAINT IF EXISTS daily_quest_templates_role_check,
    DROP CONSTRAINT IF EXISTS daily_quest_templates_xp_reward_check,
    DROP COLUMN IF EXISTS role,
    DROP COLUMN IF EXISTS xp_reward;

UPDATE daily_quest_templates SET reward_amount = CASE code
    WHEN 'DAILY_DISCOVER' THEN 5
    WHEN 'DAILY_FAVORITE' THEN 6
    WHEN 'DAILY_CONTACT' THEN 8
    WHEN 'DAILY_SELLER_STEP' THEN 10
    WHEN 'DAILY_DELIVERY' THEN 12
    ELSE reward_amount
END;

UPDATE user_daily_quests udq SET reward_amount = dqt.reward_amount
FROM daily_quest_templates dqt WHERE dqt.code = udq.template_code;

ALTER TABLE user_daily_quests DROP CONSTRAINT IF EXISTS user_daily_quests_reward_amount_check;
ALTER TABLE user_daily_quests ADD CONSTRAINT user_daily_quests_reward_amount_check CHECK (reward_amount > 0);

ALTER TABLE daily_quest_templates DROP CONSTRAINT IF EXISTS daily_quest_templates_reward_amount_check;
ALTER TABLE daily_quest_templates ADD CONSTRAINT daily_quest_templates_reward_amount_check CHECK (reward_amount > 0);
ALTER TABLE daily_quest_templates DROP CONSTRAINT IF EXISTS daily_quest_templates_action_type_check;
ALTER TABLE daily_quest_templates ADD CONSTRAINT daily_quest_templates_action_type_check
    CHECK (action_type IN ('AD_VIEWED', 'AD_FAVORITED', 'MESSAGE_SENT', 'AD_CREATED', 'DELIVERY_USED', 'REVIEW_LEFT', 'BOOKING_CREATED'));

ALTER TABLE reward_transactions DROP CONSTRAINT IF EXISTS reward_transactions_source_kind_check;
ALTER TABLE reward_transactions ADD CONSTRAINT reward_transactions_source_kind_check
    CHECK (source_kind IN ('TASK_COMPLETION', 'DAILY_QUEST', 'STREAK'));

ALTER TABLE domain_events DROP CONSTRAINT IF EXISTS domain_events_event_type_check;
ALTER TABLE domain_events ADD CONSTRAINT domain_events_event_type_check CHECK (event_type IN (
    'TASK_PROGRESS_UPDATED', 'TASK_COMPLETED', 'XP_EARNED', 'PET_LEVEL_UP', 'PET_MOOD_CHANGED',
    'ROOM_ITEM_UNLOCKED', 'STORY_STAGE_COMPLETED', 'STORY_COMPLETED', 'LEADERBOARD_SCORE_UPDATED',
    'ACHIEVEMENT_UNLOCKED', 'PET_CHARACTER_UNLOCKED', 'AVITO_REWARD_EARNED',
    'REWARD_CATALOG_UNLOCKED', 'DAILY_QUEST_UPDATED', 'DAILY_QUEST_COMPLETED', 'STREAK_UPDATED',
    'LISTING_VIEWED', 'LISTING_FAVORITED', 'SELLER_CONTACTED', 'LISTING_PUBLISHED',
    'LISTING_IMPROVED', 'LISTING_SOLD', 'DELIVERY_USED'
));
