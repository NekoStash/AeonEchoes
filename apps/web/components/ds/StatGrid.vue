<script setup lang="ts">
import { cn } from '~/lib/utils'

type StatGridColumns = 'auto' | 'two' | 'three' | 'four' | 'five'

const props = withDefaults(
  defineProps<{
    columns?: StatGridColumns
    class?: string
  }>(),
  {
    columns: 'auto'
  }
)

const columnClass = computed(() => {
  const columns: Record<StatGridColumns, string> = {
    auto: 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-4',
    two: 'grid-cols-1 sm:grid-cols-2',
    three: 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-3',
    four: 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-4',
    five: 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-5'
  }
  return columns[props.columns]
})
</script>

<template>
  <div :class="cn('grid min-w-0 gap-3', columnClass, props.class)">
    <slot />
  </div>
</template>
