CREATE TABLE listing_categories (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    sort_order SMALLINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (btrim(code) <> ''),
    CHECK (btrim(name) <> ''),
    CHECK (sort_order >= 0),
    CHECK (updated_at >= created_at)
);

CREATE TABLE listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    category_code TEXT NOT NULL REFERENCES listing_categories (code),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price_kopecks BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'DRAFT',
    is_demo BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    sold_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (btrim(title) <> ''),
    CHECK (price_kopecks >= 0),
    CHECK (status IN ('DRAFT', 'PUBLISHED', 'UNPUBLISHED', 'SOLD')),
    CHECK ((status = 'PUBLISHED') = (published_at IS NOT NULL)),
    CHECK ((status = 'SOLD') = (sold_at IS NOT NULL)),
    CHECK (updated_at >= created_at)
);

CREATE INDEX idx_listings_public_catalog
    ON listings (category_code, published_at DESC, id DESC)
    WHERE status = 'PUBLISHED';
CREATE INDEX idx_listings_owner
    ON listings (owner_id, updated_at DESC, id DESC);

CREATE TABLE listing_photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    sort_order SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (listing_id, sort_order),
    CHECK (btrim(url) <> ''),
    CHECK (sort_order >= 0)
);

CREATE TABLE listing_favorites (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    listing_id UUID NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, listing_id)
);

CREATE INDEX idx_listing_favorites_user
    ON listing_favorites (user_id, created_at DESC, listing_id);

CREATE TABLE listing_daily_views (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    listing_id UUID NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    viewed_on DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, listing_id, viewed_on)
);

CREATE TABLE listing_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    recipient_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (sender_id <> recipient_id),
    CHECK (btrim(body) <> '')
);

CREATE UNIQUE INDEX idx_listing_messages_first_buyer_contact
    ON listing_messages (listing_id, sender_id)
    WHERE sender_id <> recipient_id;
CREATE INDEX idx_listing_messages_listing
    ON listing_messages (listing_id, created_at, id);

CREATE TABLE listing_deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL UNIQUE REFERENCES listings (id) ON DELETE RESTRICT,
    buyer_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    seller_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    delivery_used BOOLEAN NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (buyer_id <> seller_id)
);

INSERT INTO listing_categories (code, name, sort_order) VALUES
    ('FURNITURE', 'Мебель и интерьер', 1),
    ('ELECTRONICS', 'Электроника', 2),
    ('AUTO', 'Авто', 3),
    ('TRAVEL', 'Путешествия', 4),
    ('REAL_ESTATE', 'Недвижимость', 5),
    ('SERVICES', 'Услуги', 6);

INSERT INTO users (id, email, password_hash, created_at, updated_at) VALUES
    ('11111111-1111-1111-1111-111111111111', 'demo.seller@avitosha.local', 'demo-account-not-for-login', NOW(), NOW()),
    ('22222222-2222-2222-2222-222222222222', 'demo.second-seller@avitosha.local', 'demo-account-not-for-login', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO listings (
    id, owner_id, category_code, title, description, price_kopecks, status, is_demo, published_at, created_at, updated_at
) VALUES
    ('10000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'FURNITURE',
     'Винтажная настольная лампа', 'Тёплая лампа для рабочего стола. Аккуратно использовалась дома, провод и выключатель исправны.', 249000, 'PUBLISHED', TRUE, NOW() - INTERVAL '3 days', NOW() - INTERVAL '4 days', NOW()),
    ('10000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'FURNITURE',
     'Деревянный письменный стол', 'Компактный стол для учёбы или работы. Есть небольшие следы использования, конструкция крепкая.', 790000, 'PUBLISHED', TRUE, NOW() - INTERVAL '2 days', NOW() - INTERVAL '3 days', NOW()),
    ('10000000-0000-0000-0000-000000000003', '22222222-2222-2222-2222-222222222222', 'ELECTRONICS',
     'Наушники для музыки и звонков', 'Лёгкие полноразмерные наушники, подходят для учёбы, видеозвонков и спокойной работы дома.', 359000, 'PUBLISHED', TRUE, NOW() - INTERVAL '1 day', NOW() - INTERVAL '2 days', NOW()),
    ('10000000-0000-0000-0000-000000000004', '22222222-2222-2222-2222-222222222222', 'SERVICES',
     'Помогу собрать мебель', 'Соберу стол, стеллаж или кресло, привезу базовый набор инструментов и подскажу по уходу.', 150000, 'PUBLISHED', TRUE, NOW() - INTERVAL '12 hours', NOW() - INTERVAL '1 day', NOW());

INSERT INTO listings (
    id, owner_id, category_code, title, description, price_kopecks, status, is_demo, published_at, created_at, updated_at
) VALUES
    ('10000000-0000-0000-0000-000000000008', '11111111-1111-1111-1111-111111111111', 'AUTO',
     'Toyota Camry 2015', 'Надёжный седан Toyota Camry 2015 года. Автомобиль в хорошем состоянии, комфортный салон, автоматическая коробка передач и уверенная динамика для города и трассы. Все подробности расскажу по телефону.', 150000000, 'PUBLISHED', TRUE, NOW() - INTERVAL '6 hours', NOW() - INTERVAL '1 day', NOW()),
    ('10000000-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', 'AUTO',
     'Mitsubishi ASX 2012', 'Компактный кроссовер Mitsubishi ASX 2012 года. Практичный автомобиль для города и поездок за город, удобная посадка, экономичный двигатель и вместительный багажник. Состояние и комплектацию уточняйте по телефону.', 108000000, 'PUBLISHED', TRUE, NOW() - INTERVAL '5 hours', NOW() - INTERVAL '1 day', NOW()),
    ('10000000-0000-0000-0000-000000000010', '11111111-1111-1111-1111-111111111111', 'TRAVEL',
     'Горящие туры на выходные', 'Подберём горящий тур на ближайшие выходные с вылетом из вашего города. В стоимость входит размещение и базовая программа отдыха, варианты направлений и актуальные даты уточняйте при обращении.', 2000000, 'PUBLISHED', TRUE, NOW() - INTERVAL '4 hours', NOW() - INTERVAL '1 day', NOW()),
    ('10000000-0000-0000-0000-000000000011', '22222222-2222-2222-2222-222222222222', 'REAL_ESTATE',
     'Квартира, 2 комнаты, 48 м²', 'Двухкомнатная квартира площадью 48 м² в новострое. Квартира без мебели, чистовая отделка и удобная планировка. Все подробности по звонку.', 600000000, 'PUBLISHED', TRUE, NOW() - INTERVAL '3 hours', NOW() - INTERVAL '1 day', NOW());

INSERT INTO listing_photos (listing_id, url, sort_order) VALUES
    ('10000000-0000-0000-0000-000000000001', 'https://storage.yandexcloud.net/avitosha-demo-images/lamp-1.jpg', 0),
    ('10000000-0000-0000-0000-000000000001', 'https://storage.yandexcloud.net/avitosha-demo-images/lamp-2.jpg', 1),
    ('10000000-0000-0000-0000-000000000001', 'https://storage.yandexcloud.net/avitosha-demo-images/lamp-3.jpg', 2),
    ('10000000-0000-0000-0000-000000000002', 'https://storage.yandexcloud.net/avitosha-demo-images/desk-1.jpg', 0),
    ('10000000-0000-0000-0000-000000000002', 'https://storage.yandexcloud.net/avitosha-demo-images/desk-2.jpg', 1),
    ('10000000-0000-0000-0000-000000000002', 'https://storage.yandexcloud.net/avitosha-demo-images/desk-3.jpg', 2),
    ('10000000-0000-0000-0000-000000000003', 'https://storage.yandexcloud.net/avitosha-demo-images/headphones-1.jpg', 0),
    ('10000000-0000-0000-0000-000000000003', 'https://storage.yandexcloud.net/avitosha-demo-images/headphones-2.jpg', 1),
    ('10000000-0000-0000-0000-000000000003', 'https://storage.yandexcloud.net/avitosha-demo-images/headphones-3.jpg', 2),
    ('10000000-0000-0000-0000-000000000004', 'https://storage.yandexcloud.net/avitosha-demo-images/service-1.webp', 0),
    ('10000000-0000-0000-0000-000000000004', 'https://storage.yandexcloud.net/avitosha-demo-images/service-2.webp', 1),
    ('10000000-0000-0000-0000-000000000008', 'https://storage.yandexcloud.net/avitosha-demo-images/camry-1.jpg', 0),
    ('10000000-0000-0000-0000-000000000008', 'https://storage.yandexcloud.net/avitosha-demo-images/camry-2.jpg', 1),
    ('10000000-0000-0000-0000-000000000008', 'https://storage.yandexcloud.net/avitosha-demo-images/camry-3.jpg', 2),
    ('10000000-0000-0000-0000-000000000009', 'https://storage.yandexcloud.net/avitosha-demo-images/mits-1.jpg', 0),
    ('10000000-0000-0000-0000-000000000009', 'https://storage.yandexcloud.net/avitosha-demo-images/mits-2.jpg', 1),
    ('10000000-0000-0000-0000-000000000009', 'https://storage.yandexcloud.net/avitosha-demo-images/mits-3.jpg', 2),
    ('10000000-0000-0000-0000-000000000010', 'https://storage.yandexcloud.net/avitosha-demo-images/travel-1.png', 0),
    ('10000000-0000-0000-0000-000000000010', 'https://storage.yandexcloud.net/avitosha-demo-images/travel-2.jpg', 1),
    ('10000000-0000-0000-0000-000000000011', 'https://storage.yandexcloud.net/avitosha-demo-images/flat-1.jpg', 0),
    ('10000000-0000-0000-0000-000000000011', 'https://storage.yandexcloud.net/avitosha-demo-images/flat-2.jpg', 1);
