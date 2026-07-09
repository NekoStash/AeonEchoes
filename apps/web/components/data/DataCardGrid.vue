<script setup lang="ts">
import { cn } from '~/lib/utils'

type GridDensity = 'compact' | 'comfortable' | 'relaxed'
type GridColumns = 'auto' | 'two' | 'three' | 'four'

type Item = Record<string, unknown>

const props = withDefaults(
  defineProps<{
    items: Item[]
    itemKey?: string
    density?: GridDensity
    columns?: GridColumns
    class?: string
  }>(),
  {
    itemKey: 'id',
    density: 'comfortable',
    columns: 'auto'
  }
)

const gridClass = computed(() => {
  const columns: Record<GridColumns, string> = {
    auto: 'grid-cols-1 md:grid-cols-2 xl:grid-cols-3',
    two: 'grid-cols-1 md:grid-cols-2',
    three: 'grid-cols-1 md:grid-cols-2 xl:grid-cols-3',
    four: 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-4'
  }
  return columns[props.columns]
})

const gapClass = computed(() => {
  const densities: Record<GridDensity, string> = {
    compact: 'gap-3',
    comfortable: 'gap-4',
    relaxed: 'gap-6'
  }
  return densities[props.density]
})

function stableItemKey(item: Item, index: number) {
  const value = item[props.itemKey]
  return typeof value === 'string' || typeof value === 'number' ? value : index
}
</script>

<template>
  <div v-if="items.length > 0" :class="cn('grid min-w-0', gridClass, gapClass, props.class)">
    <slot v-for="(item, index) in items" :key="stableItemKey(item, index)" :item="item" :index="index" />
  </div>
  <DataEmptyState v-else :class="props.class" />
</template>
