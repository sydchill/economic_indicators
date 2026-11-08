<template>
  <div>
    <AppNavbar />

    <section class="section">
      <div class="container is-max-widescreen">

        <div class="level mb-5">
          <div class="level-left">
            <div>
              <h1 class="title is-3 mb-1">Setups</h1>
              <p class="subtitle is-6 has-text-grey mb-0" v-if="lastScored">
                Differential of asset scores · as of {{ lastScored }}
              </p>
              <p class="subtitle is-6 has-text-grey mb-0" v-else-if="!loading">
                No pair scores yet
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
              class="chip"
              :class="{ 'is-active': activeCurrency === '' }"
              @click="activeCurrency = ''"
            >
              All pairs
            </button>
            <button
              v-for="ccy in availableCurrencies"
              :key="ccy"
              class="chip"
              :class="{ 'is-active': activeCurrency === ccy }"
              @click="activeCurrency = ccy"
            >
              {{ ccy }} pairs
              <span class="chip-count">{{ countFor(ccy) }}</span>
            </button>
          </div>

          <label class="checkbox-toggle">
            <input v-model="showNeutral" type="checkbox" />
            <span>Show neutral</span>
          </label>
        </div>

        <div v-if="loading && !rows.length" class="has-text-grey">Loading pairs…</div>

        <div v-else-if="!visiblePairs.length" class="box has-text-centered has-text-grey">
          No pairs match this filter.
          <template v-if="!showNeutral">
            <br /><button class="button is-small is-light mt-3" @click="showNeutral = true">
              Show neutral pairs
            </button>
          </template>
        </div>

        <table v-else class="table is-fullwidth is-hoverable scores-table">
          <thead>
            <tr>
              <th>Pair</th>
              <th class="has-text-right">Base</th>
              <th class="has-text-right">Quote</th>
              <th class="has-text-right">Differential</th>
              <th>Signal</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in visiblePairs"
              :key="row.symbol"
              :class="{ 'is-selected': selected?.symbol === row.symbol, 'is-unscored': !row.has_score }"
              @click="open(row)"
            >
              <td>
                <div class="pair-symbol">
                  <span class="pair-base">{{ row.base_currency }}</span>
                  <span class="pair-quote">{{ row.quote_currency }}</span>
                </div>
                <div class="pair-code">{{ row.symbol }}</div>
              </td>
              <td class="has-text-right num">
                <span :class="scoreColor(row.base_score)">{{ formatScore(row.base_score) }}</span>
                <span class="ccy-note">{{ row.base_currency }}</span>
              </td>
              <td class="has-text-right num">
                <span :class="scoreColor(row.quote_score)">{{ formatScore(row.quote_score) }}</span>
                <span class="ccy-note">{{ row.quote_currency }}</span>
              </td>
              <td class="has-text-right">
                <span v-if="row.has_score" class="diff-value" :class="scoreColor(row.differential)">
                  {{ formatSigned(row.differential) }}
                </span>
                <span v-else class="has-text-grey">—</span>
              </td>
              <td>
                <span v-if="row.bucket" class="bucket-tag" :class="bucketClass(row.bucket)">
                  {{ bucketLabel(row.bucket) }}
                </span>
                <span v-else class="has-text-grey is-size-7">not scored</span>
              </td>
            </tr>
          </tbody>
        </table>

      </div>
    </section>

    <!-- Drawer -->
    <transition name="fade">
      <div v-if="selected" class="drawer-overlay" @click="close" />
    </transition>

    <transition name="slide">
      <aside v-if="selected" class="drawer">
        <header class="drawer-header">
          <div>
            <div class="drawer-title">
              {{ selected.base_currency }}
              <span class="drawer-vs">vs</span>
              {{ selected.quote_currency }}
            </div>
            <div class="drawer-sub">{{ selected.symbol }}</div>
          </div>
          <div class="drawer-score">
            <div class="score-value large" :class="scoreColor(selected.differential)">
              {{ formatSigned(selected.differential) }}
            </div>
            <div v-if="selected.bucket" class="bucket-tag" :class="bucketClass(selected.bucket)">
              {{ bucketLabel(selected.bucket) }}
            </div>
          </div>
          <button class="drawer-close" aria-label="Close" @click="close">×</button>
        </header>

        <div class="drawer-body">

          <div class="math-block">
            <div class="math-side">
              <div class="math-label">Base · {{ selected.base_currency }}</div>
              <div class="math-value" :class="scoreColor(selected.base_score)">
                {{ formatScore(selected.base_score) }}
              </div>
            </div>
            <div class="math-op">−</div>
            <div class="math-side">
              <div class="math-label">Quote · {{ selected.quote_currency }}</div>
              <div class="math-value" :class="scoreColor(selected.quote_score)">
                {{ formatScore(selected.quote_score) }}
              </div>
            </div>
            <div class="math-op">=</div>
            <div class="math-side">
              <div class="math-label">Differential</div>
              <div class="math-value" :class="scoreColor(selected.differential)">
                {{ formatSigned(selected.differential) }}
              </div>
            </div>
          </div>

          <p class="math-explain">
            A positive differential means <strong>{{ selected.base_currency }}</strong> is
            fundamentally stronger than <strong>{{ selected.quote_currency }}</strong> —
            the pair is biased upward. Negative means the opposite.
          </p>

          <div v-if="detailLoading" class="has-text-grey mt-5">Loading breakdowns…</div>

          <template v-else>
            <section v-if="baseDetail" class="asset-section">
              <h3 class="asset-heading">
                <span class="heading-ccy">{{ selected.base_currency }}</span>
                <span class="heading-name">{{ baseDetail.name }}</span>
                <span class="heading-score" :class="scoreColor(baseDetail.composite_score)">
                  {{ formatScore(baseDetail.composite_score) }}
                </span>
              </h3>
              <CategoryBreakdown :categories="baseDetail.category_inputs" />
            </section>

            <hr class="section-divider" />

            <section v-if="quoteDetail" class="asset-section">
              <h3 class="asset-heading">
                <span class="heading-ccy">{{ selected.quote_currency }}</span>
                <span class="heading-name">{{ quoteDetail.name }}</span>
                <span class="heading-score" :class="scoreColor(quoteDetail.composite_score)">
                  {{ formatScore(quoteDetail.composite_score) }}
                </span>
              </h3>
              <CategoryBreakdown :categories="quoteDetail.category_inputs" />
            </section>
          </template>
        </div>
      </aside>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppNavbar from '../components/Navigation.vue'
import CategoryBreakdown from '@/components/CategoryBreakdown.vue'
import { listPairScores, getAssetScore } from '@/api/market'
import type { AssetScore, PairScore, PairBucket } from '@/types/market'

const rows = ref<PairScore[]>([])
const loading = ref(false)
const error = ref('')
const showNeutral = ref(false)
const activeCurrency = ref('')

const selected = ref<PairScore | null>(null)
const baseDetail = ref<AssetScore | null>(null)
const quoteDetail = ref<AssetScore | null>(null)
const detailLoading = ref(false)

const availableCurrencies = computed(() => {
  const set = new Set<string>()
  rows.value.forEach((p) => {
    set.add(p.base_currency)
    set.add(p.quote_currency)
  })
  return Array.from(set).sort()
})

const lastScored = computed(() => {
  const dates = rows.value.map((r) => r.scored_at).filter(Boolean) as string[]
  return dates.length ? dates.sort().pop()! : ''
})

const visiblePairs = computed(() => {
  let list = [...rows.value]

  if (activeCurrency.value) {
    list = list.filter(
      (p) => p.base_currency === activeCurrency.value || p.quote_currency === activeCurrency.value,
    )
  }

  if (!showNeutral.value) {
    list = list.filter((p) => p.bucket !== 'NEUTRAL')
  }

  list.sort((a, b) => (b.differential ?? -999) - (a.differential ?? -999))
  return list
})

function countFor(ccy: string): number {
  return rows.value.filter(
    (p) => p.base_currency === ccy || p.quote_currency === ccy,
  ).length
}

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    rows.value = await listPairScores()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e?.message ?? 'Failed to load pairs'
  } finally {
    loading.value = false
  }
}

async function open(pair: PairScore) {
  if (selected.value?.symbol === pair.symbol) {
    close()
    return
  }
  selected.value = pair
  baseDetail.value = null
  quoteDetail.value = null
  detailLoading.value = true
  try {
    const [base, quote] = await Promise.all([
      getAssetScore(pair.base_currency),
      getAssetScore(pair.quote_currency),
    ])
    baseDetail.value = base
    quoteDetail.value = quote
  } catch (e) {
    // individual detail fetch failures just leave the drawer sparser
    console.error('[setups] detail fetch failed', e)
  } finally {
    detailLoading.value = false
  }
}

function close() {
  selected.value = null
  baseDetail.value = null
  quoteDetail.value = null
}

onMounted(refresh)

// ---------- formatting ----------

function formatScore(v: number | null | undefined): string {
  if (v == null) return '—'
  return (v >= 0 ? '+' : '') + v.toFixed(2)
}

function formatSigned(v: number | null | undefined): string {
  if (v == null) return '—'
  return (v >= 0 ? '+' : '') + v.toFixed(2)
}

function scoreColor(v: number | null | undefined): string {
  if (v == null) return ''
  if (v >= 60) return 'col-very-bull'
  if (v > 0) return 'col-bull'
  if (v > -60) return 'col-bear'
  return 'col-very-bear'
}

function bucketClass(b: PairBucket): string {
  return {
    VERY_BULLISH: 'bucket-very-bull',
    BULLISH: 'bucket-bull',
    NEUTRAL: 'bucket-neutral',
    BEARISH: 'bucket-bear',
    VERY_BEARISH: 'bucket-very-bear',
  }[b]
}

function bucketLabel(b: PairBucket): string {
  return {
    VERY_BULLISH: 'Very Bullish',
    BULLISH: 'Bullish',
    NEUTRAL: 'Neutral',
    BEARISH: 'Bearish',
    VERY_BEARISH: 'Very Bearish',
  }[b]
}
</script>

<style lang="scss" scoped>
/* Toolbar */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.chips {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.35rem 0.7rem;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: transparent;
  color: var(--text-muted);
  font-size: 0.75rem;
  font-family: var(--font);
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;

  &:hover { color: var(--text); border-color: var(--text-muted); }

  &.is-active {
    color: #fff;
    background: var(--accent);
    border-color: var(--accent);
  }
}

.chip-count {
  font-size: 0.625rem;
  opacity: 0.7;
}

.checkbox-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--text-muted);
  font-size: 0.75rem;
  cursor: pointer;

  input { accent-color: var(--accent); }
}

/* Table */
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

  td { border-color: var(--border); vertical-align: middle; }

  tbody tr {
    cursor: pointer;
    transition: background 0.1s;

    &:hover { background: var(--surface); }
    &.is-selected { background: var(--surface); }
    &.is-unscored { opacity: 0.5; }
  }
}

.pair-symbol {
  display: flex;
  align-items: baseline;
  gap: 2px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-size: 0.9375rem;
  font-weight: 600;
}

.pair-base { color: var(--text); }
.pair-quote { color: var(--text-muted); }

.pair-code {
  font-size: 0.6875rem;
  color: var(--text-muted);
  letter-spacing: 0.05em;
  margin-top: 2px;
}

.num {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}

.ccy-note {
  display: block;
  font-size: 0.6875rem;
  color: var(--text-muted);
  margin-top: 2px;
  letter-spacing: 0.05em;
}

.diff-value {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-weight: 600;
  font-size: 0.9375rem;
}

.bucket-tag {
  display: inline-block;
  padding: 3px 9px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 500;

  &.bucket-very-bull { background: rgba(34, 197, 94, 0.18);  color: #4ade80; }
  &.bucket-bull      { background: rgba(34, 197, 94, 0.10);  color: #86efac; }
  &.bucket-neutral   { background: rgba(139, 144, 154, 0.15); color: #8b909a; }
  &.bucket-bear      { background: rgba(249, 115, 22, 0.12); color: #fb923c; }
  &.bucket-very-bear { background: rgba(239, 68, 68, 0.18);  color: #f87171; }
}

.col-very-bull { color: #4ade80; }
.col-bull      { color: #86efac; }
.col-bear      { color: #fb923c; }
.col-very-bear { color: #f87171; }

/* Drawer */
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
  width: min(620px, 100vw);
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
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-weight: 600;
  font-size: 1.125rem;
}

.drawer-vs {
  color: var(--text-muted);
  font-weight: 400;
  font-size: 0.875rem;
  margin: 0 0.25rem;
}

.drawer-sub {
  margin-top: 0.25rem;
  color: var(--text-muted);
  font-size: 0.8125rem;
  letter-spacing: 0.05em;
}

.drawer-score {
  margin-left: auto;
  text-align: right;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.25rem;
}

.score-value {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-weight: 600;

  &.large { font-size: 1.5rem; }
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

/* Math block */
.math-block {
  display: flex;
  align-items: center;
  justify-content: space-around;
  padding: 1rem;
  background: var(--surface-2);
  border-radius: 8px;
  margin-bottom: 0.75rem;
}

.math-side { text-align: center; }

.math-label {
  font-size: 0.6875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  margin-bottom: 0.25rem;
}

.math-value {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-weight: 600;
  font-size: 1.125rem;
}

.math-op {
  color: var(--text-muted);
  font-size: 1.125rem;
  font-weight: 300;
}

.math-explain {
  color: var(--text-muted);
  font-size: 0.8125rem;
  margin-bottom: 1.5rem;
}

/* Asset sections */
.asset-section { margin-top: 1rem; }

.asset-heading {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 1rem;
  padding-bottom: 0.5rem;
  border-bottom: 2px solid var(--border);
}

.heading-ccy {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  color: var(--text);
}

.heading-name {
  color: var(--text-muted);
  font-weight: 400;
  font-size: 0.875rem;
}

.heading-score {
  margin-left: auto;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}

.section-divider {
  border: 0;
  border-top: 1px dashed var(--border);
  margin: 2rem 0 1rem;
}

/* Transitions */
.fade-enter-active,
.fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from,
.fade-leave-to     { opacity: 0; }

.slide-enter-active,
.slide-leave-active { transition: transform 0.25s ease; }
.slide-enter-from,
.slide-leave-to     { transform: translateX(100%); }
</style>