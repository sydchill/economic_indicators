-- Re-bucket any NEUTRAL rows before restoring the 4-bucket constraint
UPDATE market.pair_scores SET bucket = 'BULLISH' WHERE bucket = 'NEUTRAL' AND differential >= 0;
UPDATE market.pair_scores SET bucket = 'BEARISH' WHERE bucket = 'NEUTRAL';

ALTER TABLE market.pair_scores
  DROP CONSTRAINT IF EXISTS pair_scores_bucket_check;

ALTER TABLE market.pair_scores
  ADD CONSTRAINT pair_scores_bucket_check
  CHECK (bucket IN ('VERY_BULLISH','BULLISH','BEARISH','VERY_BEARISH'));