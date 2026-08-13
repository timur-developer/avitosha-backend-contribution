ALTER TABLE listing_deals DROP CONSTRAINT listing_deals_listing_id_key;
ALTER TABLE listing_deals ADD CONSTRAINT listing_deals_listing_id_buyer_id_key UNIQUE (listing_id, buyer_id);
