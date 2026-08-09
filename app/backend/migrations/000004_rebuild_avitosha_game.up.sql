DROP TABLE IF EXISTS inventory_items;
DROP TABLE IF EXISTS pet_daily_states;
DROP TABLE IF EXISTS pets;

CREATE TABLE room_items (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    asset_key TEXT NOT NULL UNIQUE,
    position_key TEXT NOT NULL UNIQUE,
    unlock_level SMALLINT NOT NULL DEFAULT 1,
    sort_order SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (btrim(code) <> ''),
    CHECK (btrim(name) <> ''),
    CHECK (unlock_level BETWEEN 1 AND 5),
    CHECK (sort_order >= 0)
);

CREATE TABLE stories (
    code TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    total_stages SMALLINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (btrim(code) <> ''),
    CHECK (btrim(title) <> ''),
    CHECK (total_stages > 0)
);

CREATE TABLE pets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT 'Авитоша',
    level SMALLINT NOT NULL DEFAULT 1,
    growth_xp INTEGER NOT NULL DEFAULT 0,
    mood TEXT NOT NULL DEFAULT 'CALM',
    character TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id),
    CHECK (btrim(name) <> ''),
    CHECK (level BETWEEN 1 AND 5),
    CHECK (growth_xp >= 0),
    CHECK (mood IN ('CALM', 'CURIOUS', 'HAPPY', 'EXCITED', 'PROUD', 'SLEEPING')),
    CHECK (character IS NULL OR character IN ('EXPLORER', 'ENTREPRENEUR', 'MECHANIC', 'TRAVELER', 'ARCHITECT', 'CRAFTSPERSON')),
    CHECK (updated_at >= created_at)
);

CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    pet_phrase TEXT NOT NULL,
    action_type TEXT NOT NULL,
    category TEXT,
    target_value INTEGER NOT NULL,
    xp_reward INTEGER NOT NULL DEFAULT 0,
    avito_reward_type TEXT,
    avito_reward_amount INTEGER NOT NULL DEFAULT 0,
    room_item_code TEXT REFERENCES room_items (code),
    story_code TEXT REFERENCES stories (code),
    story_stage SMALLINT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (btrim(code) <> ''),
    CHECK (btrim(title) <> ''),
    CHECK (action_type IN ('AD_VIEWED', 'AD_FAVORITED', 'MESSAGE_SENT', 'AD_CREATED', 'DELIVERY_USED', 'REVIEW_LEFT', 'BOOKING_CREATED')),
    CHECK (target_value > 0),
    CHECK (xp_reward >= 0),
    CHECK (avito_reward_amount >= 0),
    CHECK ((story_code IS NULL) = (story_stage IS NULL)),
    CHECK (story_stage IS NULL OR story_stage > 0),
    UNIQUE (story_code, story_stage)
);

CREATE INDEX idx_tasks_match ON tasks (action_type, category) WHERE is_active;

CREATE TABLE user_story_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    story_code TEXT NOT NULL REFERENCES stories (code),
    current_stage SMALLINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, story_code),
    CHECK (current_stage >= 0),
    CHECK (status IN ('ACTIVE', 'COMPLETED')),
    CHECK ((status = 'COMPLETED') = (completed_at IS NOT NULL))
);

CREATE TABLE user_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    progress INTEGER NOT NULL DEFAULT 0,
    target_value INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    rewarded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, task_id),
    CHECK (target_value > 0),
    CHECK (progress BETWEEN 0 AND target_value),
    CHECK (status IN ('ACTIVE', 'COMPLETED', 'REWARDED', 'EXPIRED')),
    CHECK ((status IN ('COMPLETED', 'REWARDED')) = (completed_at IS NOT NULL)),
    CHECK ((status = 'REWARDED') = (rewarded_at IS NOT NULL))
);

CREATE INDEX idx_user_tasks_active ON user_tasks (user_id, status);

CREATE TABLE user_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    event_id UUID NOT NULL UNIQUE,
    action_type TEXT NOT NULL,
    entity_id TEXT,
    category TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    occurred_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ,
    result_events JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (action_type IN ('AD_VIEWED', 'AD_FAVORITED', 'MESSAGE_SENT', 'AD_CREATED', 'DELIVERY_USED', 'REVIEW_LEFT', 'BOOKING_CREATED')),
    CHECK (jsonb_typeof(metadata) = 'object'),
    CHECK (jsonb_typeof(result_events) = 'array')
);

CREATE INDEX idx_user_actions_daily ON user_actions (user_id, occurred_at);

CREATE TABLE user_room_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    item_code TEXT NOT NULL REFERENCES room_items (code),
    status TEXT NOT NULL,
    source_task_id UUID REFERENCES tasks (id),
    unlocked_at TIMESTAMPTZ NOT NULL,
    placed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, item_code),
    CHECK (status IN ('UNLOCKED', 'PLACED')),
    CHECK ((status = 'PLACED') = (placed_at IS NOT NULL))
);

CREATE TABLE weekly_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    week_start DATE NOT NULL,
    earned_xp INTEGER NOT NULL DEFAULT 0,
    completed_tasks INTEGER NOT NULL DEFAULT 0,
    completed_stages INTEGER NOT NULL DEFAULT 0,
    score INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, week_start),
    CHECK (earned_xp >= 0),
    CHECK (completed_tasks >= 0),
    CHECK (completed_stages >= 0),
    CHECK (score = earned_xp + completed_tasks * 20 + completed_stages * 50)
);

CREATE INDEX idx_weekly_progress_rank ON weekly_progress (week_start, score DESC, user_id);

CREATE TABLE daily_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    date DATE NOT NULL,
    actions_count INTEGER NOT NULL DEFAULT 0,
    completed_tasks INTEGER NOT NULL DEFAULT 0,
    earned_xp INTEGER NOT NULL DEFAULT 0,
    level_before SMALLINT NOT NULL,
    level_after SMALLINT NOT NULL,
    unlocked_room_items TEXT[] NOT NULL DEFAULT '{}',
    story_stage_before SMALLINT NOT NULL,
    story_stage_after SMALLINT NOT NULL,
    weekly_score_delta INTEGER NOT NULL DEFAULT 0,
    pet_mood TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, date),
    CHECK (actions_count >= 0),
    CHECK (completed_tasks >= 0),
    CHECK (earned_xp >= 0),
    CHECK (level_before BETWEEN 1 AND 5),
    CHECK (level_after BETWEEN 1 AND 5),
    CHECK (story_stage_before >= 0),
    CHECK (story_stage_after >= story_stage_before),
    CHECK (weekly_score_delta >= 0),
    CHECK (pet_mood IN ('CALM', 'CURIOUS', 'HAPPY', 'EXCITED', 'PROUD', 'SLEEPING'))
);

CREATE TABLE achievements (
    code TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    icon_key TEXT NOT NULL,
    sort_order SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (btrim(code) <> ''),
    CHECK (sort_order >= 0)
);

CREATE TABLE user_achievements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    achievement_code TEXT NOT NULL REFERENCES achievements (code),
    unlocked_at TIMESTAMPTZ NOT NULL,
    UNIQUE (user_id, achievement_code)
);

CREATE TABLE pet_activity_scores (
    user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    buyer_score INTEGER NOT NULL DEFAULT 0,
    seller_score INTEGER NOT NULL DEFAULT 0,
    auto_score INTEGER NOT NULL DEFAULT 0,
    travel_score INTEGER NOT NULL DEFAULT 0,
    real_estate_score INTEGER NOT NULL DEFAULT 0,
    services_score INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (buyer_score >= 0),
    CHECK (seller_score >= 0),
    CHECK (auto_score >= 0),
    CHECK (travel_score >= 0),
    CHECK (real_estate_score >= 0),
    CHECK (services_score >= 0)
);

CREATE TABLE domain_events (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    action_id UUID NOT NULL REFERENCES user_actions (id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::JSONB,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (event_type IN ('TASK_PROGRESS_UPDATED', 'TASK_COMPLETED', 'XP_EARNED', 'PET_LEVEL_UP', 'PET_MOOD_CHANGED', 'ROOM_ITEM_UNLOCKED', 'STORY_STAGE_COMPLETED', 'STORY_COMPLETED', 'LEADERBOARD_SCORE_UPDATED', 'ACHIEVEMENT_UNLOCKED', 'PET_CHARACTER_UNLOCKED')),
    CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX idx_domain_events_user_time ON domain_events (user_id, occurred_at);

INSERT INTO room_items (code, name, description, asset_key, position_key, unlock_level, sort_order) VALUES
    ('BOX', 'Коробка', 'Первая коробка Авитоши после новоселья', 'room.box', 'box', 1, 0),
    ('DESK', 'Стол', 'Рабочий стол для новой комнаты', 'room.desk', 'desk', 1, 1),
    ('LAMP', 'Лампа', 'Тёплый свет для рабочего места', 'room.lamp', 'lamp', 1, 2),
    ('CHAIR', 'Кресло', 'Удобное кресло Авитоши', 'room.chair', 'chair', 2, 3),
    ('PLANT', 'Растение', 'Зелёная деталь уютной комнаты', 'room.plant', 'plant', 2, 4),
    ('POSTER', 'Постер', 'Финальная деталь первой комнаты', 'room.poster', 'poster', 3, 5),
    ('PIGGY_BANK', 'Копилка', 'Будущий предмет за достижения', 'room.piggy_bank', 'piggy-bank', 3, 6),
    ('TOY_CAR', 'Игрушечная машина', 'Деталь характера Механика', 'room.toy_car', 'toy-car', 3, 7),
    ('SUITCASE', 'Чемодан', 'Деталь характера Путешественника', 'room.suitcase', 'suitcase', 3, 8);

INSERT INTO stories (code, title, description, total_stages) VALUES
    ('FIRST_ROOM', 'Обустроить первую комнату', 'Помоги Авитоше превратить пустую комнату в настоящий дом', 5);

INSERT INTO tasks (
    code, title, description, pet_phrase, action_type, category, target_value, xp_reward,
    avito_reward_type, avito_reward_amount, room_item_code, story_code, story_stage
) VALUES
    ('VIEW_FURNITURE_ADS', 'Помоги Авитоше выбрать стол', 'Посмотри 5 объявлений с мебелью', 'Давай найдём стол для нашего нового дома!', 'AD_VIEWED', 'FURNITURE', 5, 30, 'AVITO_BONUS', 10, 'DESK', 'FIRST_ROOM', 1),
    ('FAVORITE_FURNITURE_AD', 'Авитоша нашёл красивую лампу', 'Добавь одно объявление с мебелью в избранное', 'Сохраним эту лампу, чтобы не потерять?', 'AD_FAVORITED', 'FURNITURE', 1, 30, 'AVITO_BONUS', 10, 'LAMP', 'FIRST_ROOM', 2),
    ('MESSAGE_SELLER', 'Авитоша хочет узнать подробности', 'Напиши продавцу', 'Спросим продавца, удобно ли это кресло?', 'MESSAGE_SENT', NULL, 1, 40, 'AVITO_BONUS', 15, 'CHAIR', 'FIRST_ROOM', 3),
    ('CREATE_FIRST_AD', 'Помоги Авитоше стать продавцом', 'Размести первое объявление', 'Освободим место и найдём дом ненужной вещи!', 'AD_CREATED', NULL, 1, 50, 'AVITO_BONUS', 20, 'PLANT', 'FIRST_ROOM', 4),
    ('USE_DELIVERY', 'Последняя деталь комнаты', 'Воспользуйся Авито Доставкой', 'Постер уже едет — осталось выбрать доставку!', 'DELIVERY_USED', NULL, 1, 80, 'AVITO_BONUS', 25, 'POSTER', 'FIRST_ROOM', 5);

INSERT INTO achievements (code, title, description, icon_key, sort_order) VALUES
    ('FIRST_STEP', 'Первый шаг', 'Выполнить первое задание', 'achievement.first_step', 1),
    ('HOUSEWARMING', 'Новоселье', 'Открыть первый предмет комнаты', 'achievement.housewarming', 2),
    ('EXPLORER', 'Исследователь', 'Посмотреть 5 объявлений', 'achievement.explorer', 3),
    ('IN_TOUCH', 'На связи', 'Написать продавцу', 'achievement.in_touch', 4),
    ('FIRST_AD', 'Первое объявление', 'Разместить объявление', 'achievement.first_ad', 5),
    ('ROOM_COMPLETE', 'Комната готова', 'Завершить первую сюжетную линию', 'achievement.room_complete', 6);
