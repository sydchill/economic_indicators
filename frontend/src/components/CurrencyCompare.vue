<template>
   <div id="currency_compare">
      <Navigation />
      <div class="box m-6">
        <div class="is-flex mb-4">
          <h1 class="has-text-weight-medium">Select currency pair</h1>
        </div>
        <div class="is-flex">
         <div class="select is-link mr-3">
            <select v-model="currency1" @change="compareEconomicIndicators">
              <option v-for="c in currencyList">{{ c }}</option>
            </select>
          </div>
          <div class="select is-link mr-3">
            <select v-model="currency2" @change="compareEconomicIndicators">
              <option v-for="c in currencyList">{{ c }}</option>
            </select>
          </div>
        </div>

        <table class="table mt-6 is-fullwidth">
          <thead>
            <tr>
              <th>Indicator</th>
              <th></th>
              <th></th>
              <th></th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="indicator in indicators">
              <td>{{ indicator }}</td>
              <td>{{ currency1Data[indicator] }}</td>
              <td>{{ currency2Data[indicator] }}</td>
              <td></td>
              <td></td>
            </tr>
            <tr>
              <td>Score</td>
              <td>{{ currency1total }}</td>
              <td>{{ currency2total }}</td>
              <td></td>
              <td></td>
            </tr>
          </tbody>
        </table>
        <div class="is-flex mt-6">
          <h1 class="has-text-info-dark">{{ bias }}</h1>
        </div>
      </div>
   </div>
</template>


<script setup lang="ts">
import { computed, reactive, ref, toRefs } from 'vue';
import Navigation from './Navigation.vue';
import currencyData from './currency_compare.json'

interface CurrencyData {
  [key: string] : number | string
}
const currencyList = ref<string[]>(currencyData.map(e => { return e.Country}));

let data = reactive({
  currency1: 'USA' as string,
  currency2: 'EUR' as string,
  bias: "" as string,
  currency1total: 0 as number,
  currency2total: 0 as number,
  
})

// COMPUTED

let currency1Data = computed((): CurrencyData => {

  return currencyData.filter( e => { return e.Country === currency1.value})[0]
})

let currency2Data = computed((): CurrencyData => {

return currencyData.filter( e => { return e.Country === currency2.value})[0]
})


let indicators = computed((): string[] => {

  return Object.keys(currencyData[0])
})


// METHODS

let compareEconomicIndicators =  () => {
  const indicators = [
    { key: 'GDP Growth Rate', higherIsBetter: true },
    { key: 'Unemployment Rate', higherIsBetter: false },
    { key: 'Inflation Rate', higherIsBetter: true },
    { key: 'Interest Rate', higherIsBetter: true },
    { key: 'Business Confidence', higherIsBetter: true },
    { key: 'Manufacturing PMI', higherIsBetter: true },
    { key: 'Consumer Confidence', higherIsBetter: true },
    { key: 'Retail Sales MoM', higherIsBetter: true },
    { key: 'GDP Annual Growth Rate', higherIsBetter: true },
    { key: 'Current Account', higherIsBetter: true }
  ]

  let currency1Score = 0;
  let currency2Score = 0;

    // Compare each indicator
    indicators.forEach(indicator => {
    const key = indicator.key as keyof CurrencyData;
    const currency1Value = currency1Data.value[key] as number;
    const currency2Value = currency2Data.value[key] as number;

    if (currency1Value !== undefined && currency2Value !== undefined) {
      if (currency1Value > currency2Value && indicator.higherIsBetter) {
        currency1Score++;
      } else if (currency1Value < currency2Value && !indicator.higherIsBetter) {
        currency1Score++;
      }  else {
        currency2Score++;
      }
    }
  });
  currency1total.value = currency1Score
  currency2total.value = currency2Score

  // Determine the result
  if (currency1Score > currency2Score) {
    bias.value = `${currency1Data.value.Country} is more bullish compared to ${currency2Data.value.Country}.`;
  } else if (currency1Score < currency2Score) {
    bias.value = `${currency2Data.value.Country} is more bullish compared to ${currency1Data.value.Country}.`;
  } else {
    bias.value = `${currency1Data.value.Country} and ${currency2Data.value.Country} are neutral relative to each other.`;
  }
}


let {currency1, currency2, bias, currency1total, currency2total} = toRefs(data)
</script>

<style lang="scss" scoped>
@import "../assets/scss/main";

.box {
  border-radius: 10px !important;
  color: $primary-color !important;
  background-color: #F7F8FF;
  box-shadow: rgba(50, 50, 93, 0.25) 0px 13px 27px -5px, rgba(0, 0, 0, 0.3) 0px 8px 16px -8px;
}

.table {
  background-color: #F7F8FF !important;
  th, td {
    color: $primary-color !important;
  }
  
}

</style>
