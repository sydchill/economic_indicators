// src/api/types.ts

export interface MacroSnapshotRaw {
  Country: string;
  // plus all your raw indicator fields if you ever need them on frontend
  [key: string]: any;
}

export interface ScoreBreakdown {
  country: string;
  total_score: number;
  components: Record<string, number>;
  explanation: string;
  raw_indicators: MacroSnapshotRaw;
}

export interface PairSentiment {
  base: string;
  quote: string;
  base_score: number;
  quote_score: number;
  pair_score: number;
  explanation: string;
  base_details: ScoreBreakdown;
  quote_details: ScoreBreakdown;
}

export interface InstrumentScore {
  symbol: string;
  asset_type: "index" | "metal" | string;
  total_score: number;
  components: Record<string, number>;
  explanation: string;
}
