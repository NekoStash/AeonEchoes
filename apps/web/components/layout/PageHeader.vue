<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    eyebrow?: string
    title: string
    description?: string
    align?: 'start' | 'center'
    class?: string
  }>(),
  {
    align: 'start'
  }
)
</script>

<template>
  <header :class="cn('flex min-w-0 flex-col gap-4 lg:flex-row lg:items-start lg:justify-between', align === 'center' && 'text-center lg:text-left', props.class)">
    <div class="min-w-0 flex-1">
      <p v-if="eyebrow" class="truncate text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">{{ eyebrow }}</p>
      <h1 class="mt-1 max-w-5xl break-words text-2xl font-semibold tracking-tight text-foreground md:text-3xl">{{ title }}</h1>
      <p v-if="description" class="mt-2 max-w-4xl break-words text-sm leading-7 text-muted-foreground">{{ description }}</p>
      <slot />
    </div>
    <div v-if="$slots.actions" class="flex w-full min-w-0 flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:justify-end lg:shrink-0">
      <slot name="actions" />
    </div>
  </header>
</template>
