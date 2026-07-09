<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    eyebrow?: string
    title?: string
    description?: string
    divided?: boolean
    class?: string
  }>(),
  {
    divided: true
  }
)
</script>

<template>
  <header :class="cn('flex min-w-0 flex-col gap-3 px-4 py-4 sm:flex-row sm:items-start sm:justify-between sm:px-5', divided && 'border-b border-border', props.class)">
    <div class="min-w-0 flex-1">
      <p v-if="eyebrow" class="truncate text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{{ eyebrow }}</p>
      <h2 v-if="title" class="mt-1 break-words text-base font-semibold text-foreground">{{ title }}</h2>
      <p v-if="description" class="mt-1 break-words text-sm leading-6 text-muted-foreground">{{ description }}</p>
      <slot />
    </div>
    <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center gap-2 sm:justify-end">
      <slot name="actions" />
    </div>
  </header>
</template>
