import { http } from './http'
import type { AssetScore } from '@/types/market'

export async function listAssetScores(): Promise<AssetScore[]> {
  const { data } = await http.get<AssetScore[]>('/api/market/assets')
  return data
}

export async function getAssetScore(code: string): Promise<AssetScore> {
  const { data } = await http.get<AssetScore>(`/api/market/assets/${code}`)
  return data
}