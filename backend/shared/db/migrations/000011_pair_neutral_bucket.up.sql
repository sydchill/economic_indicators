-- Add NEUTRAL to pair_scores only. asset_scores keeps its 4-bucket CHECK.
ALTER TABLE market.pair_scores
  DROP CONSTRAINT IF EXISTS pair_scores_bucket_check;

ALTER TABLE market.pair_scores
  ADD CONSTRAINT pair_scores_bucket_check
  CHECK (bucket IN ('VERY_BULLISH','BULLISH','NEUTRAL','BEARISH','VERY_BEARISH'));