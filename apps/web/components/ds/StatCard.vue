<script setup lang="ts">
import { cn } from '~/lib/utils'

type StatTone = 'info' | 'success' | 'warning' | 'danger' | 'neutral'

const props = withDefaults(
  defineProps<{
    label: string
    value: string | number
    hint?: string
    tone?: StatTone
    class?: string
  }>(),
  {
    hint: undefined,
    tone: 'neutral'
  }
)

const toneClass = computed(() => {
  const tones: Record<StatTone, string> = {
    info: 'border-state-info-border bg-state-info-surface text-state-info-foreground',
    success: 'border-state-success-border bg-state-success-surface text-state-success-foreground',
    warning: 'border-state-warning-border bg-state-warning-surface text-state-warning-foreground',
    danger: 'border-state-danger-border bg-state-danger-surface text-state-danger-foreground',
    neutral: 'border-border bg-surface text-foreground'
  }
  return tones[props.tone]
})
</script>

<template>
  <article :class="cn('min-w-0 rounded-2xl border p-4 shadow-sm', toneClass, props.class)">
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0">
        <p class="truncate text-xs font-medium uppercase tracking-[0.18em] opacity-70">{{ label }}</p>
        <p class="mt-3 break-words text-2xl font-semibold text-current">{{ value }}</p>
      </div>
      <div v-if="$slots.icon" class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-current/10">
        <slot name="icon" />
      </div>
    </div>
    <p v-if="hint" class="mt-2 text-xs leading-5 opacity-70">{{ hint }}</p>
    <slot />
  </article>
</template>
