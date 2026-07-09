<script setup lang="ts">
import { cn } from '~/lib/utils'

type FilterBarDensity = 'compact' | 'normal' | 'relaxed'

const props = withDefaults(
  defineProps<{
    density?: FilterBarDensity
    class?: string
  }>(),
  {
    density: 'normal'
  }
)

const densityClass = computed(() => {
  const densities: Record<FilterBarDensity, string> = {
    compact: 'gap-2 p-2',
    normal: 'gap-3 p-3',
    relaxed: 'gap-4 p-4'
  }
  return densities[props.density]
})
</script>

<template>
  <div :class="cn('surface-muted flex min-w-0 flex-col rounded-2xl sm:flex-row sm:flex-wrap sm:items-center', densityClass, props.class)">
    <div v-if="$slots.search" class="min-w-0 flex-1 sm:min-w-64">
      <slot name="search" />
    </div>
    <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2">
      <slot />
    </div>
    <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center justify-end gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
