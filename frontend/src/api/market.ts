import { http } from './http'
import type { AssetScore, PairScore } from '@/types/market'

export async function listAssetScores(): Promise<AssetScore[]> {
  const { data } = await http.get<AssetScore[]>('/api/market/assets')
  return data
}

export async function getAssetScore(code: string): Promise<AssetScore> {
  const { data } = await http.get<AssetScore>(`/api/market/assets/${code}`)
  return data
}

export async function listPairScores(currency?: string): Promise<PairScore[]> {
  const params = currency ? { currency } : undefined
  const { data } = await http.get<PairScore[]>('/api/market/pairs', { params })
  return data
}

export async function getPairScore(symbol: string): Promise<PairScore> {
  const { data } = await http.get<PairScore>(`/api/market/pairs/${symbol}`)
  return data
}