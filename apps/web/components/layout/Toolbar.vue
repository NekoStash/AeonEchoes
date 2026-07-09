<script setup lang="ts">
import { cn } from '~/lib/utils'

type ToolbarDensity = 'compact' | 'normal' | 'relaxed'

const props = withDefaults(
  defineProps<{
    density?: ToolbarDensity
    wrap?: boolean
    class?: string
  }>(),
  {
    density: 'normal',
    wrap: true
  }
)

const densityClass = computed(() => {
  const densities: Record<ToolbarDensity, string> = {
    compact: 'gap-2 p-2',
    normal: 'gap-3 p-3',
    relaxed: 'gap-4 p-4'
  }
  return densities[props.density]
})
</script>

<template>
  <div :class="cn('surface-muted flex min-w-0 items-center rounded-2xl', wrap ? 'flex-wrap' : 'overflow-x-auto subtle-scrollbar', densityClass, props.class)">
    <div v-if="$slots.start" class="flex min-w-0 flex-1 flex-wrap items-center gap-2">
      <slot name="start" />
    </div>
    <slot />
    <div v-if="$slots.end" class="ml-auto flex shrink-0 flex-wrap items-center justify-end gap-2">
      <slot name="end" />
    </div>
  </div>
</template>
