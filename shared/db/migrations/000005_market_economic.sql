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