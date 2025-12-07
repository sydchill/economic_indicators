// src/api/macro.ts
import { apiClient } from "./clients";

export interface ScoreBreakdown {
  country: string;
  total_score: number;
  components: Record<string, number>;
  explanation: string;
  raw_indicators: any;
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
  asset_type: string;
  total_score: number;
  components: Record<string, number>;
  explanation: string;
}

type MacroScoresResponse = {
  data: Record<string, ScoreBreakdown>;
};

type InstrumentsResponse = {
  data: Record<string, InstrumentScore>;
};

export async function fetchMacroScores(): Promise<Record<string, ScoreBreakdown>> {
  const res = await apiClient.get<MacroScoresResponse>("/macro/scores");
  return res.data.data;
}

export async function fetchPairSentiment(base: string, quote: string): Promise<PairSentiment> {
  const res = await apiClient.get<PairSentiment>("/macro/pair", {
    params: { base, quote },
  });
  return res.data;
}

export async function fetchInstrumentScores(): Promise<Record<string, InstrumentScore>> {
  const res = await apiClient.get<InstrumentsResponse>("/instruments/scores");
  return res.data.data;
}
