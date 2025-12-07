<template>
  <div class="container pt-4">
    <div class="box">
      <div class="is-flex is-align-items-center">
        <div><span class="has-text-weight-bold has-text-white">MacroSentiments</span></div>
        <div class="ml-6">
          <div class="tabs">
            <ul>
              <li :class="{ 'is-active': activeTab === 'overview' }">
                <a @click="activeTab = 'overview'">
                  Overview
                </a>
              </li>
              <li :class="{ 'is-active': activeTab === 'fx' }">
                <a  @click="activeTab = 'fx'">
                  FX Pairs
                </a>
              </li>
              <li :class="{ 'is-active': activeTab === 'instruments' }">
                <a @click="activeTab = 'instruments'">
                  Indices & Metals
                </a>
              </li>
            </ul>
          </div>
        </div>
      </div>
      <div class="mt-6">
        <!-- Overview tab -->
        <div v-if="activeTab === 'overview'">
          <h2 class="title is-5">Currency Overview</h2>

          <!-- 🔥 Strongest / Weakest row -->
          <div class="box mb-4">
            <div class="columns">
              <div class="column">
                <h3 class="subtitle is-6 has-text-weight-semibold">Top Strongest</h3>
                <div class="tags">
                  <span
                    v-for="item in strongestCurrencies"
                    :key="`strong-${item.code}`"
                    class="tag is-medium"
                    :class="scoreTagClass(item.score.total_score)"
                  >
                    {{ item.code }} ({{ item.score.total_score.toFixed(2) }})
                  </span>
                </div>
              </div>

              <div class="column">
                <h3 class="subtitle is-6 has-text-weight-semibold">Top Weakest</h3>
                <div class="tags">
                  <span
                    v-for="item in weakestCurrencies"
                    :key="`weak-${item.code}`"
                    class="tag is-medium"
                    :class="scoreTagClass(item.score.total_score)"
                  >
                    {{ item.code }} ({{ item.score.total_score.toFixed(2) }})
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- your existing currency grid can stay below this -->
          <div class="columns is-multiline">
            <div
              class="column is-4"
              v-for="(score, code) in macroScores"
              :key="code"
            >
              <div class="box">
                <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
                  <span class="tag is-info is-light">{{ code }}</span>
                  <span class="tag" :class="scoreTagClass(score.total_score)">
                    {{ score.total_score.toFixed(2) }}
                  </span>
                </div>

                <p class="is-size-7 has-text-grey">
                  {{ score.explanation }}
                </p>
                       <!-- optional: quick view of key components -->
                <div class="is-size-7">
                  <p>
                    <strong>GDP:</strong>
                    {{ (score.components.gdp_growth ?? 0).toFixed(2) }}
                    &nbsp;|&nbsp;
                    <strong>Unemp:</strong>
                    {{ (score.components.unemployment ?? 0).toFixed(2) }}
                  </p>
                  <p>
                    <strong>Inflation:</strong>
                    {{ (score.components.inflation ?? 0).toFixed(2) }}
                    &nbsp;|&nbsp;
                    <strong>Rates:</strong>
                    {{ (score.components.interest_rate ?? 0).toFixed(2) }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- FX Pairs tab -->
        <div v-if="activeTab === 'fx'">
          <h2 class="title is-5">FX Pair Sentiment</h2>

          <div class="columns is-align-items-flex-end">
            <div class="column is-4">
              <div class="field">
                <label class="label">Base Currency</label>
                <div class="control">
                  <div class="select is-fullwidth">
                    <select v-model="base" @change="onPairChange">
                      <option v-for="(score, code) in macroScores" :key="code" :value="code">
                        {{ code }}
                      </option>
                    </select>
                  </div>
                </div>
              </div>

              <div class="field">
                <label class="label">Quote Currency</label>
                <div class="control">
                  <div class="select is-fullwidth">
                    <select v-model="quote" @change="onPairChange">
                      <option v-for="(score, code) in macroScores" :key="code" :value="code">
                        {{ code }}
                      </option>
                    </select>
                  </div>
                </div>
              </div>
            </div>
            <div class="column is-8">
              <div class="box">
                <h3 class="title is-5">{{ pair?.base }}/{{ pair?.quote }}</h3>
                <p><strong>Pair Score:</strong> {{ pair?.pair_score }}</p>
                <p class="mt-2 is-size-7">{{ pair?.explanation }}</p>
              </div>
            </div>
          </div>
        </div>
        <!-- Indices & Metals tab -->
        <div v-if="activeTab === 'instruments'">
          <h2 class="title is-4">Indices & Metals</h2>

          <div class="columns is-multiline">
            <div 
              class="column is-4"
              v-for="inst in instruments" 
              :key="inst.symbol"
            >
              <div class="box indice_metal_box">
                <h3 class="title is-5">{{ inst.symbol }} ({{ inst.asset_type }})</h3>
                <p><strong>Score:</strong> {{ inst.total_score }}</p>
                <p class="mt-2 is-size-7">{{ inst.explanation }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>


<script setup lang="ts">
// src/App.vue <script setup>
import { ref, computed, onMounted } from "vue";
import {
  fetchMacroScores,
  fetchPairSentiment,
  fetchInstrumentScores,
  type ScoreBreakdown,
  type PairSentiment,
  type InstrumentScore,
} from "../api/macro";

const macroScores = ref<Record<string, ScoreBreakdown>>({});
const instruments = ref<Record<string, InstrumentScore>>({});
const pair = ref<PairSentiment | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);
const activeTab = ref<"overview" | "fx" | "instruments">("overview");

const base = ref("GBP");
const quote = ref("USD");

// 🔹 convert macroScores object → sorted array
const sortedCurrencies = computed(() => {
  return Object.entries(macroScores.value)
    .map(([code, score]) => ({ code, score }))
    .sort((a, b) => b.score.total_score - a.score.total_score);
});

// 🔹 top 3 strongest
const strongestCurrencies = computed(() => sortedCurrencies.value.slice(0, 3));

// 🔹 top 3 weakest
const weakestCurrencies = computed(() =>
  [...sortedCurrencies.value].reverse().slice(0, 3)
);

// 🔹 color helper for tags
function scoreTagClass(score: number) {
  if (score > 0.25) return "is-success";
  if (score < -0.3) return "is-danger";
  return "is-warning";
}

// existing load logic
async function loadAll() {
  try {
    loading.value = true;
    error.value = null;

    const [scores, inst, pairData] = await Promise.all([
      fetchMacroScores(),
      fetchInstrumentScores(),
      fetchPairSentiment(base.value, quote.value),
    ]);

    macroScores.value = scores;
    instruments.value = inst;
    pair.value = pairData;
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  } finally {
    loading.value = false;
  }
}

function onPairChange() {
  fetchPairSentiment(base.value, quote.value)
    .then((res: any) => (pair.value = res))
    .catch((e: any) => (error.value = e?.message ?? String(e)));
}

onMounted(loadAll);

</script>

<style lang="scss" scoped>
@import "../assets/scss/main";
.box {
  background-color: #0F172A ;
}

.container {
  min-height: 100vh;
}

.indice_metal_box {
  height: 15rem;
  color: #f1f1e6;
  border-color: #538ff8 !important;
}
</style>
