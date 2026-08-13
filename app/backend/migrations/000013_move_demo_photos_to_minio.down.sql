UPDATE listing_photos
SET url = REPLACE(
    url,
    '/storage/avitosha-photos/demo/',
    'https://storage.yandexcloud.net/avitosha-demo-images/'
)
WHERE url LIKE '/storage/avitosha-photos/demo/%';
