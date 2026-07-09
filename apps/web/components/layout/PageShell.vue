<script setup lang="ts">
import { cn } from '~/lib/utils'

type PageDensity = 'compact' | 'normal' | 'relaxed'

const props = withDefaults(
  defineProps<{
    as?: string
    density?: PageDensity
    class?: string
  }>(),
  {
    as: 'section',
    density: 'normal'
  }
)

const densityClass = computed(() => {
  const densities: Record<PageDensity, string> = {
    compact: 'space-y-density-lg',
    normal: 'space-y-density-xl',
    relaxed: 'space-y-8'
  }
  return densities[props.density]
})
</script>

<template>
  <component :is="as" :class="cn('min-w-0', densityClass, props.class)">
    <slot />
  </component>
</template>
