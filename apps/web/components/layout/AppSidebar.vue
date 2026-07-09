<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    label?: string
    class?: string
    contentClass?: string
  }>(),
  {
    label: 'Primary navigation'
  }
)
</script>

<template>
  <aside
    :aria-label="label"
    :class="cn('hidden w-sidebar shrink-0 border-r border-border bg-surface text-surface-foreground lg:flex lg:flex-col', props.class)"
  >
    <div :class="cn('flex min-h-0 flex-1 flex-col', props.contentClass)">
      <slot />
    </div>
  </aside>
</template>
