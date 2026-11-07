CREATE TABLE market.pair_scores (
    id           BIGSERIAL PRIMARY KEY,
    pair_id      BIGINT NOT NULL REFERENCES market.pairs(id) ON DELETE CASCADE,
    scored_at    DATE NOT NULL,
    base_score   NUMERIC(7,4) NOT NULL,
    quote_score  NUMERIC(7,4) NOT NULL,
    differential NUMERIC(7,4) NOT NULL,
    bucket       TEXT NOT NULL CHECK (bucket IN ('VERY_BULLISH','BULLISH','BEARISH','VERY_BEARISH')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (pair_id, scored_at)
);

CREATE INDEX idx_pair_scores_date      ON market.pair_scores (scored_at DESC, differential DESC);
CREATE INDEX idx_pair_scores_pair_date ON market.pair_scores (pair_id, scored_at DESC);