<template>
  <div v-if="!hasAny" class="has-text-grey">No category data available.</div>

  <section
    v-for="(cat, category) in categories"
    v-else
    :key="category"
    class="cat-block"
  >
    <div class="cat-header">
      <span class="cat-name">{{ prettyCategory(String(category)) }}</span>
      <div class="cat-header-right">
        <span class="cat-meta">
          {{ cat.active_indicators }}/{{ cat.total_indicators }} active ·
          w {{ cat.weight_renormalized.toFixed(2) }}
        </span>
        <span class="cat-score" :class="scoreColor(cat.score)">
          {{ formatSigned(cat.score) }}
        </span>
      </div>
    </div>

    <table class="input-table">
      <thead>
        <tr>
          <th>Indicator</th>
          <th class="has-text-right">Value</th>
          <th class="has-text-right">Signal</th>
          <th class="has-text-right">Contrib</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(ind, i) in cat.indicators"
          :key="i"
          :class="{ 'is-dropped': ind.mode === 'dropped' }"
        >
          <td>
            <div class="ind-name">{{ prettyIndicator(ind.slug) }}</div>
            <div class="ind-meta">
              <template v-if="ind.mode === 'dropped'">
                dropped · {{ ind.drop_reason ?? 'no data' }}
              </template>
              <template v-else>
                {{ ind.mode }} · dir {{ ind.direction > 0 ? '↑' : '↓' }} ·
                w {{ ind.weight_original.toFixed(2) }}
                <template v-if="ind.raw_change != null">
                  · Δ {{ formatValue(ind.raw_change) }}
                </template>
              </template>
            </div>
          </td>
          <td class="has-text-right num">{{ formatValue(ind.value) }}</td>
          <td
            class="has-text-right num"
            :class="netSignal(ind) >= 0 ? 'pos-text' : 'neg-text'"
          >
            {{ ind.mode === 'dropped' ? '—' : formatSigned(netSignal(ind)) }}
          </td>
          <td
            class="has-text-right num"
            :class="contribution(ind) >= 0 ? 'pos-text' : 'neg-text'"
          >
            {{ ind.mode === 'dropped' ? '—' : formatSigned(contribution(ind)) }}
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CategoryInput, IndicatorDetail } from '@/types/market'

const props = defineProps<{
  categories: Record<string, CategoryInput> | null | undefined
}>()

const hasAny = computed(() => {
  const c = props.categories
  return c && typeof c === 'object' && Object.keys(c).length > 0
})

function formatSigned(v: number | null | undefined): string {
  if (v == null || Number.isNaN(v)) return '—'
  return (v >= 0 ? '+' : '') + v.toFixed(2)
}

function formatValue(v: number | null | undefined): string {
  if (v == null || Number.isNaN(v)) return '—'
  if (Math.abs(v) >= 1000) return v.toLocaleString(undefined, { maximumFractionDigits: 0 })
  return v.toLocaleString(undefined, { maximumFractionDigits: 2 })
}

function scoreColor(v: number | null | undefined): string {
  if (v == null) return ''
  if (v >= 60) return 'col-very-bull'
  if (v > 0) return 'col-bull'
  if (v > -60) return 'col-bear'
  return 'col-very-bear'
}

function netSignal(ind: IndicatorDetail): number {
  return ind.signal * ind.direction
}

function contribution(ind: IndicatorDetail): number {
  return ind.signal * ind.direction * ind.weight_original
}

const CATEGORY_LABELS: Record<string, string> = {
  cot: 'COT Positioning',
  growth: 'Economic Growth',
  labour: 'Labour Market',
  inflation: 'Inflation',
  central_bank: 'Central Bank',
}

function prettyCategory(cat: string): string {
  return CATEGORY_LABELS[cat] ?? cat.charAt(0).toUpperCase() + cat.slice(1)
}

const INDICATOR_LABELS: Record<string, string> = {
  cpi_yoy: 'CPI YoY',
  cpi_mom: 'CPI MoM',
  pce_yoy: 'PCE YoY',
  ppi_yoy: 'PPI YoY',
  pmi_manufacturing: 'Manufacturing PMI',
  pmi_services: 'Services PMI',
  gdp_qoq: 'GDP QoQ',
  retail_sales_mom: 'Retail Sales MoM',
  nfp: 'Non-Farm Payrolls',
  unemployment_rate: 'Unemployment Rate',
  initial_claims: 'Initial Claims',
  continuing_claims: 'Continuing Claims',
  adp_employment_change: 'ADP Employment',
  claimant_count_change: 'Claimant Count Change',
  employment_change_thousands: 'Employment Change (K)',
  employment_change_qoq: 'Employment Change QoQ',
  employment_change_yoy: 'Employment Change YoY',
  participation_rate: 'Participation Rate',
  employment_rate: 'Employment Rate',
  wage_growth_yoy: 'Wage Growth YoY',
  job_openings: 'Job Openings',
  jobs_to_applications: 'Jobs to Applications',
  job_vacancy_rate: 'Job Vacancy Rate',
  consumer_confidence: 'Consumer Confidence',
  policy_rate: 'Policy Rate',
  noncommercial_net: 'Non-Commercial Net',
}

function prettyIndicator(slug: string | undefined): string {
  if (!slug) return '(unknown)'
  const key = slug.split('.').pop() ?? slug
  return INDICATOR_LABELS[key] ?? key.replace(/_/g, ' ')
}
</script>

<style lang="scss" scoped>
.cat-block {
  margin-bottom: 1.75rem;

  &:last-child { margin-bottom: 0; }
}

.cat-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 0.5rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--border);
}

.cat-header-right {
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
}

.cat-name {
  font-weight: 600;
  font-size: 0.9375rem;
}

.cat-meta {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-size: 0.6875rem;
  color: var(--text-muted);
}

.cat-score {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-size: 0.9375rem;
  font-weight: 600;
}

.input-table {
  width: 100%;
  border-collapse: collapse;

  th {
    font-size: 0.6875rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
    font-weight: 500;
    padding: 0.25rem 0.25rem 0.5rem;
  }

  td {
    padding: 0.4rem 0.25rem;
    border-top: 1px solid var(--border);
    font-size: 0.8125rem;
    vertical-align: top;
  }

  tr.is-dropped td {
    opacity: 0.55;
  }

  tr.is-dropped .ind-meta {
    color: var(--text-muted);
    font-style: italic;
  }
}

.ind-name {
  font-weight: 500;
  color: var(--text);
}

.ind-meta {
  margin-top: 2px;
  color: var(--text-muted);
  font-size: 0.6875rem;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}

.num {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  color: var(--text-muted);
}

.pos-text { color: #4ade80; }
.neg-text { color: #f87171; }

.col-very-bull { color: #4ade80; }
.col-bull      { color: #86efac; }
.col-bear      { color: #fb923c; }
.col-very-bear { color: #f87171; }
</style>