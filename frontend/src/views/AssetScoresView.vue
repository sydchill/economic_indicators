<template>
  <div>
    <AppNavbar />

    <section class="section">
      <div class="container is-max-widescreen">

        <div class="level mb-5">
          <div class="level-left">
            <div>
              <h1 class="title is-3 mb-1">Asset Scores</h1>
              <p class="subtitle is-6 has-text-grey mb-0" v-if="lastScored">
                Scored as of {{ lastScored }}
              </p>
              <p class="subtitle is-6 has-text-grey mb-0" v-else-if="!loading">
                No scores yet — run <code class="inline">go run ./backend/worker/cmd/score --apply</code>
              </p>
            </div>
          </div>
          <div class="level-right">
            <button
              class="button is-small is-light"
              :class="{ 'is-loading': loading }"
              @click="refresh"
            >
              Refresh
            </button>
          </div>
        </div>

        <p v-if="error" class="notification is-danger is-light">{{ error }}</p>

        <div class="toolbar mb-4">
          <div class="chips">
            <button
              v-for="f in filters"
              :key="f.key"
              class="chip"
              :class="{ 'is-active': activeFilter === f.key }"
              @click="activeFilter = f.key"
            >
              {{ f.label }}
              <span class="chip-count">{{ countOf(f.key) }}</span>
            </button>
          </div>

          <div class="sort">
            <span class="sort-label">Sort</span>
            <select v-model="sortKey" class="select-input">
              <option value="score-desc">Score (high → low)</option>
              <option value="score-asc">Score (low → high)</option>
              <option value="name">Name (A → Z)</option>
              <option value="class">Asset class</option>
            </select>
          </div>
        </div>

        <div v-if="loading && !rows.length" class="has-text-grey">Loading scores…</div>

        <div v-else-if="!filteredRows.length" class="box has-text-centered has-text-grey">
          No assets match this filter.
        </div>

        <table v-else class="table is-fullwidth is-hoverable scores-table">
          <thead>
            <tr>
              <th>Asset</th>
              <th>Class</th>
              <th class="has-text-right">Score</th>
              <th>Signal</th>
              <th class="has-text-right">Categories</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in filteredRows"
              :key="row.code"
              :class="{ 'is-selected': selected?.code === row.code, 'is-unscored': !row.has_score }"
              @click="open(row)"
            >
              <td>
                <div class="asset">
                  <span class="asset-code">{{ row.code }}</span>
                  <span class="asset-name">{{ row.name }}</span>
                </div>
              </td>
              <td>
                <span class="class-tag" :class="`class-${row.category}`">
                  {{ row.category }}
                </span>
              </td>
              <td class="has-text-right">
                <span v-if="row.has_score" class="score-value" :class="scoreColor(row.composite_score)">
                  {{ formatScore(row.composite_score) }}
                </span>
                <span v-else class="has-text-grey">—</span>
              </td>
              <td>
                <span v-if="row.bucket" class="bucket-tag" :class="bucketClass(row.bucket)">
                  {{ bucketLabel(row.bucket) }}
                </span>
                <span v-else class="has-text-grey is-size-7">not scored</span>
              </td>
              <td class="has-text-right">
                <div v-if="row.category_scores" class="cat-mini">
                  <span
                    v-for="(val, cat) in row.category_scores"
                    :key="cat"
                    class="cat-pip"
                    :class="val >= 0 ? 'pos' : 'neg'"
                    :title="`${cat}: ${val.toFixed(2)}`"
                  />
                </div>
              </td>
            </tr>
          </tbody>
        </table>

      </div>
    </section>

    <!-- Detail drawer -->
    <transition name="fade">
      <div v-if="selected" class="drawer-overlay" @click="selected = null" />
    </transition>

    <transition name="slide">
      <aside v-if="selected" class="drawer">
        <header class="drawer-header">
          <div>
            <div class="drawer-title">
              {{ selected.code }}
              <span class="class-tag" :class="`class-${selected.category}`">{{ selected.category }}</span>
            </div>
            <div class="drawer-sub">{{ selected.name }}</div>
          </div>
          <div class="drawer-score">
            <div class="score-value large" :class="scoreColor(selected.composite_score)">
              {{ formatScore(selected.composite_score) }}
            </div>
            <div v-if="selected.bucket" class="bucket-tag" :class="bucketClass(selected.bucket)">
              {{ bucketLabel(selected.bucket) }}
            </div>
          </div>
          <button class="drawer-close" aria-label="Close" @click="selected = null">×</button>
        </header>

        <div class="drawer-body">
          <div v-if="!selected.category_inputs" class="has-text-grey">
            No detail available for this asset.
          </div>

      <section v-for="(cat, category) in selected.category_inputs" :key="category" class="cat-block">
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
                        {{ ind.mode }} ·
                        dir {{ ind.direction > 0 ? '↑' : '↓' }} ·
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
        </div>
      </aside>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppNavbar from '../components/Navigation.vue'
import { listAssetScores } from '@/api/market'
import type { AssetScore, Bucket, IndicatorDetail } from '@/types/market'

const rows = ref<AssetScore[]>([])
const loading = ref(false)
const error = ref('')
const selected = ref<AssetScore | null>(null)

type FilterKey = 'all' | 'currency' | 'metal' | 'index'
const activeFilter = ref<FilterKey>('all')
const filters: { key: FilterKey; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'currency', label: 'Currencies' },
  { key: 'metal', label: 'Metals' },
  { key: 'index', label: 'Indices' },
]

type SortKey = 'score-desc' | 'score-asc' | 'name' | 'class'
const sortKey = ref<SortKey>('score-desc')

    
const lastScored = computed(() => {
  const dates = rows.value.map((r) => r.scored_at).filter(Boolean) as string[]
  if (!dates.length) return ''
  return dates.sort().pop()!
})

const filteredRows = computed(() => {
  let list = [...rows.value]
  if (activeFilter.value !== 'all') {
    list = list.filter((r) => r.category === activeFilter.value)
  }
  switch (sortKey.value) {
    case 'score-desc':
      list.sort((a, b) => (b.composite_score ?? -999) - (a.composite_score ?? -999))
      break
    case 'score-asc':
      list.sort((a, b) => (a.composite_score ?? 999) - (b.composite_score ?? 999))
      break
    case 'name':
      list.sort((a, b) => a.name.localeCompare(b.name))
      break
    case 'class':
      list.sort((a, b) =>
        a.category === b.category
          ? a.code.localeCompare(b.code)
          : a.category.localeCompare(b.category),
      )
      break
  }
  return list
})

function countOf(key: FilterKey) {
  if (key === 'all') return rows.value.length
  return rows.value.filter((r) => r.category === key).length
}

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    rows.value = await listAssetScores()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e?.message ?? 'Failed to load scores'
  } finally {
    loading.value = false
  }
}

function open(row: AssetScore) {
  selected.value = selected.value?.code === row.code ? null : row
}

onMounted(refresh)

// ---------- formatting helpers ----------
function netSignal(ind: IndicatorDetail): number {
  return ind.signal * ind.direction
}

function contribution(ind: IndicatorDetail): number {
  return ind.signal * ind.direction * ind.weight_original
}

function formatScore(v: number | null | undefined): string {
  if (v == null) return '—'
  return (v >= 0 ? '+' : '') + v.toFixed(2)
}

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

function bucketClass(b: Bucket): string {
  return {
    VERY_BULLISH: 'bucket-very-bull',
    BULLISH: 'bucket-bull',
    BEARISH: 'bucket-bear',
    VERY_BEARISH: 'bucket-very-bear',
  }[b]
}

function bucketLabel(b: Bucket): string {
  return {
    VERY_BULLISH: 'Very Bullish',
    BULLISH: 'Bullish',
    BEARISH: 'Bearish',
    VERY_BEARISH: 'Very Bearish',
  }[b]
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

function prettyIndicator(code?: string | unknown): string {
  const s = typeof code === 'string' ? code : ''
  if (!s) return '(unknown)'
  if (s.startsWith('RATES_')) {
    return s.replace('RATES_', '').replace('Y', 'Y Yield')
  }
  const stripped = s.replace(/^[A-Z]{3}_/, '')
  return INDICATOR_LABELS[stripped] ?? stripped.replace(/_/g, ' ')
}

</script>

<style lang="scss" scoped>
/* ---------- Toolbar ---------- */
.cat-header-right {
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
}

.cat-meta {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-size: 0.6875rem;
  color: var(--text-muted);
}

.input-table tr.is-dropped td {
  opacity: 0.55;
}

.input-table tr.is-dropped .ind-meta {
  color: var(--text-muted);
  font-style: italic;
}
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.chips {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.35rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: transparent;
  color: var(--text-muted);
  font-size: 0.8125rem;
  font-family: var(--font);
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;

  &:hover {
    color: var(--text);
    border-color: var(--text-muted);
  }

  &.is-active {
    color: #fff;
    background: var(--accent);
    border-color: var(--accent);
  }
}

.chip-count {
  font-size: 0.6875rem;
  opacity: 0.7;
}

.sort {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.sort-label {
  font-size: 0.8125rem;
  color: var(--text-muted);
}

.select-input {
  padding: 0.35rem 0.6rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-2);
  color: var(--text);
  font-family: var(--font);
  font-size: 0.8125rem;
  cursor: pointer;
}

/* ---------- Table ---------- */

.scores-table {
  background: transparent;

  th {
    color: var(--text-muted);
    font-size: 0.75rem;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-color: var(--border);
  }

  td {
    border-color: var(--border);
    vertical-align: middle;
  }

  tbody tr {
    cursor: pointer;
    transition: background 0.1s;

    &:hover { background: var(--surface); }
    &.is-selected { background: var(--surface); }
    &.is-unscored { opacity: 0.5; }
  }
}

.asset {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
}

.asset-code {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-weight: 600;
  color: var(--text);
}

.asset-name {
  color: var(--text-muted);
  font-size: 0.8125rem;
}

.class-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 0.6875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 500;

  &.class-currency { background: rgba(79, 140, 247, 0.15); color: #7baaf8; }
  &.class-metal    { background: rgba(234, 179, 8, 0.15);  color: #facc15; }
  &.class-index    { background: rgba(168, 85, 247, 0.15); color: #c084fc; }
}

.score-value {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-weight: 600;
  font-size: 0.9375rem;

  &.large { font-size: 1.5rem; }
}

.bucket-tag {
  display: inline-block;
  padding: 3px 9px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 500;

  &.bucket-very-bull { background: rgba(34, 197, 94, 0.18);  color: #4ade80; }
  &.bucket-bull      { background: rgba(34, 197, 94, 0.10);  color: #86efac; }
  &.bucket-bear      { background: rgba(249, 115, 22, 0.12); color: #fb923c; }
  &.bucket-very-bear { background: rgba(239, 68, 68, 0.18);  color: #f87171; }
}

.col-very-bull { color: #4ade80; }
.col-bull      { color: #86efac; }
.col-bear      { color: #fb923c; }
.col-very-bear { color: #f87171; }

.cat-mini {
  display: inline-flex;
  gap: 3px;
}

.cat-pip {
  display: inline-block;
  width: 6px;
  height: 14px;
  border-radius: 1px;

  &.pos { background: rgba(34, 197, 94, 0.7); }
  &.neg { background: rgba(239, 68, 68, 0.7); }
}

/* ---------- Drawer ---------- */

.drawer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 40;
}

.drawer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  width: min(560px, 100vw);
  background: var(--surface);
  border-left: 1px solid var(--border);
  z-index: 50;
  display: flex;
  flex-direction: column;
}

.drawer-header {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border);
}

.drawer-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-weight: 600;
  font-size: 1.125rem;
}

.drawer-sub {
  margin-top: 0.25rem;
  color: var(--text-muted);
  font-size: 0.8125rem;
}

.drawer-score {
  margin-left: auto;
  text-align: right;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.25rem;
}

.drawer-close {
  background: transparent;
  border: 0;
  color: var(--text-muted);
  font-size: 1.5rem;
  line-height: 1;
  cursor: pointer;
  padding: 0 0.5rem;
  align-self: flex-start;

  &:hover { color: var(--text); }
}

.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 1.25rem 1.5rem 2rem;
}

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

.cat-name {
  font-weight: 600;
  font-size: 0.9375rem;
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

/* ---------- Transitions ---------- */

.fade-enter-active,
.fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from,
.fade-leave-to     { opacity: 0; }

.slide-enter-active,
.slide-leave-active { transition: transform 0.25s ease; }
.slide-enter-from,
.slide-leave-to     { transform: translateX(100%); }
</style>
