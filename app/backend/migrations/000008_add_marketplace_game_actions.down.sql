DROP TABLE IF EXISTS listing_quality_awards;
DROP TABLE IF EXISTS marketplace_game_requests;
DROP TABLE IF EXISTS product_action_rules;
ALTER TABLE domain_events DROP CONSTRAINT IF EXISTS domain_events_event_type_check;
ALTER TABLE domain_events ADD CONSTRAINT domain_events_event_type_check CHECK (event_type IN ('TASK_PROGRESS_UPDATED', 'TASK_COMPLETED', 'XP_EARNED', 'PET_LEVEL_UP', 'PET_MOOD_CHANGED', 'ROOM_ITEM_UNLOCKED', 'STORY_STAGE_COMPLETED', 'STORY_COMPLETED', 'LEADERBOARD_SCORE_UPDATED', 'ACHIEVEMENT_UNLOCKED', 'PET_CHARACTER_UNLOCKED'));
ALTER TABLE pet_activity_scores DROP COLUMN IF EXISTS quality_score;
ALTER TABLE user_actions DROP CONSTRAINT IF EXISTS user_actions_action_type_check;
ALTER TABLE user_actions ADD CONSTRAINT user_actions_action_type_check CHECK (action_type IN ('AD_VIEWED', 'AD_FAVORITED', 'MESSAGE_SENT', 'AD_CREATED', 'DELIVERY_USED', 'REVIEW_LEFT', 'BOOKING_CREATED'));
