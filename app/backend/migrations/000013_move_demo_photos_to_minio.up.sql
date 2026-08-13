UPDATE listing_photos
SET url = REPLACE(
    url,
    'https://storage.yandexcloud.net/avitosha-demo-images/',
    '/storage/avitosha-photos/demo/'
)
WHERE url LIKE 'https://storage.yandexcloud.net/avitosha-demo-images/%';
