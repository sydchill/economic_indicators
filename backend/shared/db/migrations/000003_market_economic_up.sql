CREATE SCHEMA IF NOT EXISTS market;

-- Registry of every indicator the system knows about.
-- Slug is the wire format (used in snapshots); te_name is the TradingEconomics label
-- for audit only — NOT a lookup key (two slugs share a te_name: employment_change_qoq
-- and employment_change_thousands both map to TE's "Employment Change").
CREATE TABLE market.economic_indicators (
    id          SERIAL PRIMARY KEY,
    slug        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    te_name     TEXT NOT NULL,
    category    TEXT NOT NULL,
    unit        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_economic_indicators_category ON market.economic_indicators (category);

-- One row per (indicator, currency, period).
-- previous_value is the "Previous" column from the release — which reflects the
-- *revised* prior reading, not the originally reported value. That's intentional.
CREATE TABLE market.economic_data (
    id              BIGSERIAL PRIMARY KEY,
    indicator_id    INT  NOT NULL REFERENCES market.economic_indicators(id) ON DELETE RESTRICT,
    currency        TEXT NOT NULL,
    period          DATE NOT NULL,
    value           NUMERIC NOT NULL,
    previous_value  NUMERIC,
    forecast_value  NUMERIC,
    release_date    DATE NOT NULL,
    source          TEXT NOT NULL DEFAULT 'manual',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (indicator_id, currency, period)
);

CREATE INDEX idx_economic_data_lookup   ON market.economic_data (indicator_id, currency, period DESC);
CREATE INDEX idx_economic_data_release  ON market.economic_data (release_date DESC);
CREATE INDEX idx_economic_data_currency ON market.economic_data (currency);

CREATE TABLE market.assets (
    id          BIGSERIAL PRIMARY KEY,
    code        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    category    TEXT NOT NULL CHECK (category IN ('currency', 'index', 'commodity', 'crypto')),
    currency    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_assets_currency ON market.assets (currency);
CREATE INDEX idx_assets_category ON market.assets (category);

CREATE TABLE market.pairs (
    id              BIGSERIAL PRIMARY KEY,
    symbol          TEXT NOT NULL UNIQUE,
    base_currency   TEXT NOT NULL,
    quote_currency  TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pairs_distinct CHECK (base_currency <> quote_currency)
);
CREATE UNIQUE INDEX idx_pairs_currencies ON market.pairs (base_currency, quote_currency);
CREATE INDEX idx_pairs_base  ON market.pairs (base_currency);
CREATE INDEX idx_pairs_quote ON market.pairs (quote_currency);

-- 000007_market_scoring_config.up.sql  (revised)

CREATE TABLE market.asset_category_weights (
    id          BIGSERIAL PRIMARY KEY,
    asset_id    BIGINT NOT NULL REFERENCES market.assets(id) ON DELETE CASCADE,
    category    TEXT NOT NULL CHECK (category IN ('growth','labour','inflation','central_bank','sentiment','cot')),
    weight      NUMERIC(6,4) NOT NULL CHECK (weight >= 0 AND weight <= 1),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (asset_id, category)
);
CREATE INDEX idx_acw_asset ON market.asset_category_weights (asset_id);

CREATE TABLE market.asset_indicator_config (
    id            BIGSERIAL PRIMARY KEY,
    asset_id      BIGINT   NOT NULL REFERENCES market.assets(id) ON DELETE CASCADE,
    indicator_id  INT      NOT NULL REFERENCES market.economic_indicators(id) ON DELETE RESTRICT,
    category      TEXT     NOT NULL CHECK (category IN ('growth','labour','inflation','central_bank','sentiment','cot')),
    direction     SMALLINT NOT NULL CHECK (direction IN (-1, 1)),
    weight        NUMERIC(6,4) NOT NULL CHECK (weight >= 0 AND weight <= 1),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (asset_id, indicator_id)
);
CREATE INDEX idx_aic_asset     ON market.asset_indicator_config (asset_id);
CREATE INDEX idx_aic_indicator ON market.asset_indicator_config (indicator_id);
CREATE INDEX idx_aic_category  ON market.asset_indicator_config (asset_id, category);

CREATE TABLE market.asset_scores (
    id                BIGSERIAL PRIMARY KEY,
    asset_id          BIGINT NOT NULL REFERENCES market.assets(id) ON DELETE CASCADE,
    scored_at         TIMESTAMPTZ NOT NULL,
    composite_score   NUMERIC(7,4) NOT NULL,
    bucket            TEXT NOT NULL CHECK (bucket IN ('VERY_BULLISH', 'BULLISH', 'BEARISH', 'VERY_BEARISH')),
    category_scores   JSONB NOT NULL,
    category_inputs   JSONB NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (asset_id, scored_at)
);
CREATE INDEX idx_as_asset_time ON market.asset_scores (asset_id, scored_at DESC);
CREATE INDEX idx_as_bucket     ON market.asset_scores (bucket);

CREATE TABLE market.cot_data (
    id                  BIGSERIAL PRIMARY KEY,
    currency            TEXT NOT NULL,
    report_date         DATE NOT NULL,
    noncommercial_long  BIGINT NOT NULL,
    noncommercial_short BIGINT NOT NULL,
    noncommercial_net   BIGINT NOT NULL,
    source              TEXT NOT NULL DEFAULT 'cftc',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (currency, report_date)
);
CREATE INDEX idx_cot_currency_date ON market.cot_data (currency, report_date DESC);

CREATE TABLE market.retail_sentiment (
    id          BIGSERIAL PRIMARY KEY,
    pair_id     BIGINT NOT NULL REFERENCES market.pairs(id) ON DELETE CASCADE,
    snapshot_at DATE NOT NULL,
    long_pct    NUMERIC(5,2) NOT NULL CHECK (long_pct BETWEEN 0 AND 100),
    short_pct   NUMERIC(5,2) NOT NULL CHECK (short_pct BETWEEN 0 AND 100),
    source      TEXT NOT NULL DEFAULT 'manual',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (pair_id, snapshot_at)
);
CREATE INDEX idx_retail_pair_date ON market.retail_sentiment (pair_id, snapshot_at DESC);