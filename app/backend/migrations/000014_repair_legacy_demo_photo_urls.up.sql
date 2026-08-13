UPDATE listing_photos
SET url = REPLACE(
    url,
    'https://images.example.test/demo/',
    '/storage/avitosha-photos/demo/'
)
WHERE url LIKE 'https://images.example.test/demo/%';

UPDATE listing_photos
SET url = REPLACE(
    url,
    'https://storage.yandexcloud.net/avitosha-demo-images/',
    '/storage/avitosha-photos/demo/'
)
WHERE url LIKE 'https://storage.yandexcloud.net/avitosha-demo-images/%';

UPDATE listing_photos
SET url = '/storage/avitosha-photos/demo/service-1.webp'
WHERE url = '/storage/avitosha-photos/demo/service-1.jpg';
