-- =========================================
-- 1. SCHEMA
-- =========================================

CREATE SCHEMA IF NOT EXISTS market;

-- =========================================
-- 2. ASSETS (currencies, metals, indices)
-- =========================================

CREATE TABLE market.assets (
    code              TEXT PRIMARY KEY CHECK (length(code) BETWEEN 2 AND 10),
    name              TEXT NOT NULL,
    display_name      TEXT NOT NULL,
    asset_class       TEXT NOT NULL CHECK (asset_class IN ('currency','metal','index')),
    primary_currency  TEXT REFERENCES market.assets(code),
    is_major          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed currencies first (metal/index rows self-reference USD/JPY).
INSERT INTO market.assets (code, name, display_name, asset_class, is_major) VALUES
    ('USD', 'US Dollar',           'US Dollar',           'currency', TRUE),
    ('EUR', 'Euro',                'Euro',                'currency', TRUE),
    ('GBP', 'British Pound',       'British Pound',       'currency', TRUE),
    ('JPY', 'Japanese Yen',        'Japanese Yen',        'currency', TRUE),
    ('AUD', 'Australian Dollar',   'Australian Dollar',   'currency', TRUE),
    ('NZD', 'New Zealand Dollar',  'New Zealand Dollar',  'currency', TRUE),
    ('CAD', 'Canadian Dollar',     'Canadian Dollar',     'currency', TRUE),
    ('CHF', 'Swiss Franc',         'Swiss Franc',         'currency', TRUE);

INSERT INTO market.assets (code, name, display_name, asset_class, primary_currency, is_major) VALUES
    ('XAU',    'Gold',        'Gold Spot',       'metal', 'USD', TRUE),
    ('XAG',    'Silver',      'Silver Spot',     'metal', 'USD', TRUE),
    ('NAS100', 'Nasdaq 100',  'Nasdaq 100 CFD',  'index', 'USD', TRUE),
    ('SPX500', 'S&P 500',     'S&P 500 CFD',     'index', 'USD', TRUE),
    ('JP225',  'Nikkei 225',  'Nikkei 225 CFD',  'index', 'JPY', TRUE);

-- =========================================
-- 3. FX PAIRS (28 standard, in strength order)
-- =========================================

CREATE TABLE market.pairs (
    code            TEXT PRIMARY KEY CHECK (length(code) = 6),
    base_currency   TEXT NOT NULL REFERENCES market.assets(code),
    quote_currency  TEXT NOT NULL REFERENCES market.assets(code),
    display_name    TEXT NOT NULL,
    is_major        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (base_currency <> quote_currency)
);

CREATE INDEX idx_pairs_base  ON market.pairs (base_currency);
CREATE INDEX idx_pairs_quote ON market.pairs (quote_currency);

-- Convention: EUR > GBP > AUD > NZD > USD > CAD > CHF > JPY
INSERT INTO market.pairs (code, base_currency, quote_currency, display_name, is_major) VALUES
    -- EUR base (7)
    ('EURUSD','EUR','USD','EUR/USD',TRUE),
    ('EURGBP','EUR','GBP','EUR/GBP',FALSE),
    ('EURAUD','EUR','AUD','EUR/AUD',FALSE),
    ('EURNZD','EUR','NZD','EUR/NZD',FALSE),
    ('EURCAD','EUR','CAD','EUR/CAD',FALSE),
    ('EURCHF','EUR','CHF','EUR/CHF',FALSE),
    ('EURJPY','EUR','JPY','EUR/JPY',FALSE),
    -- GBP base (6)
    ('GBPUSD','GBP','USD','GBP/USD',TRUE),
    ('GBPAUD','GBP','AUD','GBP/AUD',FALSE),
    ('GBPNZD','GBP','NZD','GBP/NZD',FALSE),
    ('GBPCAD','GBP','CAD','GBP/CAD',FALSE),
    ('GBPCHF','GBP','CHF','GBP/CHF',FALSE),
    ('GBPJPY','GBP','JPY','GBP/JPY',FALSE),
    -- AUD base (5)
    ('AUDUSD','AUD','USD','AUD/USD',TRUE),
    ('AUDNZD','AUD','NZD','AUD/NZD',FALSE),
    ('AUDCAD','AUD','CAD','AUD/CAD',FALSE),
    ('AUDCHF','AUD','CHF','AUD/CHF',FALSE),
    ('AUDJPY','AUD','JPY','AUD/JPY',FALSE),
    -- NZD base (4)
    ('NZDUSD','NZD','USD','NZD/USD',TRUE),
    ('NZDCAD','NZD','CAD','NZD/CAD',FALSE),
    ('NZDCHF','NZD','CHF','NZD/CHF',FALSE),
    ('NZDJPY','NZD','JPY','NZD/JPY',FALSE),
    -- USD base (3)
    ('USDCAD','USD','CAD','USD/CAD',TRUE),
    ('USDCHF','USD','CHF','USD/CHF',TRUE),
    ('USDJPY','USD','JPY','USD/JPY',TRUE),
    -- CAD base (2)
    ('CADCHF','CAD','CHF','CAD/CHF',FALSE),
    ('CADJPY','CAD','JPY','CAD/JPY',FALSE),
    -- CHF base (1)
    ('CHFJPY','CHF','JPY','CHF/JPY',FALSE);

-- =========================================
-- 4. ECONOMIC INDICATORS (13 per currency = 104 rows)
-- =========================================

CREATE TABLE market.economic_indicators (
    id              SERIAL PRIMARY KEY,
    code            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    currency_code   TEXT NOT NULL REFERENCES market.assets(code),
    category        TEXT NOT NULL CHECK (category IN ('growth','inflation','jobs')),
    importance      TEXT NOT NULL CHECK (importance IN ('high','medium','low')),
    unit            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_econ_ind_currency ON market.economic_indicators (currency_code, category);

DO $$
DECLARE
    ccy TEXT;
    ind RECORD;
BEGIN
    FOR ccy IN SELECT unnest(ARRAY['USD','EUR','GBP','JPY','AUD','NZD','CAD','CHF']) LOOP
        FOR ind IN
            SELECT * FROM (VALUES
                ('GDP_QOQ',          'GDP Growth QoQ',        'growth',    'medium', '%'),
                ('MFG_PMI',          'Manufacturing PMI',     'growth',    'high',   'index'),
                ('SVC_PMI',          'Services PMI',          'growth',    'high',   'index'),
                ('RETAIL_SALES_MOM', 'Retail Sales MoM',      'growth',    'medium', '%'),
                ('CONSUMER_CONF',    'Consumer Confidence',   'growth',    'medium', 'index'),
                ('CPI_YOY',          'CPI YoY',               'inflation', 'high',   '%'),
                ('PPI_YOY',          'PPI YoY',               'inflation', 'medium', '%'),
                ('PCE_YOY',          'PCE YoY',               'inflation', 'high',   '%'),
                ('NFP',              'Non-Farm Payrolls',     'jobs',      'high',   'K'),
                ('UNEMPLOYMENT',     'Unemployment Rate',     'jobs',      'high',   '%'),
                ('JOBLESS_CLAIMS',   'Weekly Jobless Claims', 'jobs',      'medium', 'K'),
                ('ADP',              'ADP Employment Change', 'jobs',      'medium', 'K'),
                ('JOLTS',            'JOLTS Job Openings',    'jobs',      'medium', 'K')
            ) AS t(suffix, name, category, importance, unit)
        LOOP
            INSERT INTO market.economic_indicators
                (code, name, currency_code, category, importance, unit)
            VALUES
                (ccy || '_' || ind.suffix, ind.name, ccy, ind.category, ind.importance, ind.unit);
        END LOOP;
    END LOOP;
END $$;

-- =========================================
-- 5. ECONOMIC DATA (actual vs forecast history)
-- =========================================

CREATE TABLE market.economic_data (
    id              BIGSERIAL PRIMARY KEY,
    indicator_id    INTEGER NOT NULL REFERENCES market.economic_indicators(id),
    currency_code   TEXT NOT NULL REFERENCES market.assets(code),
    release_time    TIMESTAMPTZ NOT NULL,
    period_date     DATE,
    actual_value    NUMERIC,
    forecast_value  NUMERIC,
    previous_value  NUMERIC,
    revised_value   NUMERIC,
    surprise        NUMERIC GENERATED ALWAYS AS (actual_value - forecast_value) STORED,
    source          TEXT NOT NULL DEFAULT 'biquote',
    raw_payload     JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (indicator_id, release_time)
);

CREATE INDEX idx_econ_data_currency_time ON market.economic_data (currency_code, release_time DESC);
CREATE INDEX idx_econ_data_indicator     ON market.economic_data (indicator_id, release_time DESC);

-- =========================================
-- 6. BOND YIELDS
-- =========================================

CREATE TABLE market.bond_yields (
    id              BIGSERIAL PRIMARY KEY,
    currency_code   TEXT NOT NULL REFERENCES market.assets(code),
    maturity_years  SMALLINT NOT NULL CHECK (maturity_years IN (2,5,10,30)),
    record_date     DATE NOT NULL,
    yield_percent   NUMERIC(6,4) NOT NULL,
    source          TEXT NOT NULL DEFAULT 'fxmacrodata',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (currency_code, maturity_years, record_date)
);

CREATE INDEX idx_bond_yields_lookup ON market.bond_yields (currency_code, maturity_years, record_date DESC);

-- =========================================
-- 7. COT DATA (weekly, manual CSV import)
-- =========================================

CREATE TABLE market.cot_data (
    id                BIGSERIAL PRIMARY KEY,
    asset_id          TEXT NOT NULL REFERENCES market.assets(code),
    report_date       DATE NOT NULL,
    released_at       DATE NOT NULL,
    long_contracts    INTEGER NOT NULL,
    short_contracts   INTEGER NOT NULL,
    net_position      INTEGER NOT NULL,
    weekly_change     INTEGER,
    open_interest     INTEGER,
    source            TEXT NOT NULL DEFAULT 'cftc',
    raw_payload       JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (asset_id, report_date)
);

CREATE INDEX idx_cot_asset_date ON market.cot_data (asset_id, report_date DESC);

-- =========================================
-- 8. RETAIL SENTIMENT (pair-level, manual first)
-- =========================================

CREATE TABLE market.retail_sentiment (
    id           BIGSERIAL PRIMARY KEY,
    pair_code    TEXT NOT NULL REFERENCES market.pairs(code),
    timeframe    TEXT NOT NULL CHECK (timeframe IN ('4H','1D')),
    long_pct     NUMERIC(5,2) NOT NULL CHECK (long_pct BETWEEN 0 AND 100),
    short_pct    NUMERIC(5,2) NOT NULL CHECK (short_pct BETWEEN 0 AND 100),
    source       TEXT NOT NULL,
    snapshot_at  TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (abs(long_pct + short_pct - 100) < 0.01),
    UNIQUE (pair_code, timeframe, snapshot_at)
);

CREATE INDEX idx_sentiment_pair ON market.retail_sentiment (pair_code, snapshot_at DESC);

-- =========================================
-- 9. SCORING CONFIG (option C — per-asset)
-- =========================================

CREATE TABLE market.asset_category_weights (
    asset_id    TEXT NOT NULL REFERENCES market.assets(code),
    category    TEXT NOT NULL CHECK (category IN ('cot','growth','inflation','jobs','rates')),
    weight      NUMERIC(5,4) NOT NULL CHECK (weight >= 0 AND weight <= 1),
    params      JSONB NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (asset_id, category)
);

CREATE TABLE market.asset_indicator_config (
    asset_id      TEXT NOT NULL REFERENCES market.assets(code),
    indicator_id  INTEGER NOT NULL REFERENCES market.economic_indicators(id),
    category      TEXT NOT NULL,
    direction     SMALLINT NOT NULL CHECK (direction IN (-1, 1)),
    weight        NUMERIC(5,4) NOT NULL CHECK (weight >= 0 AND weight <= 1),
    PRIMARY KEY (asset_id, indicator_id)
);

CREATE INDEX idx_aic_asset_cat ON market.asset_indicator_config (asset_id, category);

-- =========================================
-- 10. ASSET SCORES (output — what screen 1 reads)
-- =========================================

CREATE TABLE market.asset_scores (
    asset_id          TEXT NOT NULL REFERENCES market.assets(code),
    computed_at       TIMESTAMPTZ NOT NULL,
    composite_score   NUMERIC(6,2) NOT NULL CHECK (composite_score BETWEEN -100 AND 100),
    signal            TEXT NOT NULL CHECK (signal IN ('VERY_BULLISH','BULLISH','BEARISH','VERY_BEARISH')),
    category_scores   JSONB NOT NULL,
    category_inputs   JSONB NOT NULL,
    PRIMARY KEY (asset_id, computed_at)
);

CREATE INDEX idx_asset_scores_latest ON market.asset_scores (asset_id, computed_at DESC);