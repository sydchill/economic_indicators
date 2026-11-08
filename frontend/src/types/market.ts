export type AssetClass = 'currency' | 'metal' | 'index'
export type Bucket = 'VERY_BULLISH' | 'BULLISH' | 'BEARISH' | 'VERY_BEARISH'


export interface IndicatorDetail {
  mode: 'trend' | 'cot' | 'dropped' | string
  slug: string                          
  value: number
  period: string                  
  signal: number                    
  direction: number                       
  weight_original: number                
  raw_change?: number
  previous_value?: number
  drop_reason?: string
}

export interface CategoryInput {
  score: number
  category: string
  weight_original: number
  weight_renormalized: number
  total_indicators: number
  active_indicators: number
  indicators: IndicatorDetail[]
}

export interface AssetScore {
  code: string
  name: string
  category: 'currency' | 'commodity' | 'index'
  currency?: string
  composite_score: number | null
  bucket: Bucket | null
  scored_at: string | null
  category_scores: Record<string, number> | null
  category_inputs: Record<string, CategoryInput> | null
  has_score: boolean
}

export type PairBucket = 'VERY_BULLISH' | 'BULLISH' | 'NEUTRAL' | 'BEARISH' | 'VERY_BEARISH'

export interface PairScore {
  symbol: string
  base_currency: string
  quote_currency: string
  base_score: number | null
  quote_score: number | null
  differential: number | null
  bucket: PairBucket | null
  scored_at: string | null
  has_score: boolean
}