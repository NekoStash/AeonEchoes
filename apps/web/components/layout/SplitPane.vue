<script setup lang="ts">
import { cn } from '~/lib/utils'

type SplitRatio = 'equal' | 'sidebar' | 'detail' | 'wide'

const props = withDefaults(
  defineProps<{
    ratio?: SplitRatio
    reverse?: boolean
    stickyAside?: boolean
    class?: string
    asideClass?: string
    mainClass?: string
  }>(),
  {
    ratio: 'sidebar',
    reverse: false,
    stickyAside: false
  }
)

const gridClass = computed(() => {
  const ratios: Record<SplitRatio, string> = {
    equal: 'lg:grid-cols-2',
    sidebar: 'lg:grid-cols-[minmax(0,22rem)_minmax(0,1fr)]',
    detail: 'lg:grid-cols-[minmax(0,1fr)_minmax(0,24rem)]',
    wide: 'xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]'
  }
  return ratios[props.ratio]
})
</script>

<template>
  <div :class="cn('grid min-w-0 gap-4 lg:gap-6', gridClass, reverse && 'lg:[&>*:first-child]:order-2 lg:[&>*:last-child]:order-1', props.class)">
    <aside :class="cn('min-w-0', stickyAside && 'lg:sticky lg:top-[calc(var(--layout-height-topbar)+1rem)] lg:self-start', props.asideClass)">
      <slot name="aside" />
    </aside>
    <div :class="cn('min-w-0', props.mainClass)">
      <slot />
    </div>
  </div>
</template>
