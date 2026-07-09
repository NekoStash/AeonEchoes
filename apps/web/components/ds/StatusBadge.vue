<script setup lang="ts">
import { cn } from '~/lib/utils'

type StatusTone = 'info' | 'success' | 'warning' | 'danger' | 'neutral' | 'muted'

const props = withDefaults(
  defineProps<{
    tone?: StatusTone
    pulse?: boolean
    class?: string
  }>(),
  {
    tone: 'neutral',
    pulse: false
  }
)

const toneClass = computed(() => {
  const tones: Record<StatusTone, string> = {
    info: 'border-state-info-border bg-state-info-surface text-state-info-foreground',
    success: 'border-state-success-border bg-state-success-surface text-state-success-foreground',
    warning: 'border-state-warning-border bg-state-warning-surface text-state-warning-foreground',
    danger: 'border-state-danger-border bg-state-danger-surface text-state-danger-foreground',
    neutral: 'border-border bg-surface text-foreground',
    muted: 'border-border bg-muted text-muted-foreground'
  }
  return tones[props.tone]
})

const dotClass = computed(() => {
  const tones: Record<StatusTone, string> = {
    info: 'bg-state-info',
    success: 'bg-state-success',
    warning: 'bg-state-warning',
    danger: 'bg-state-danger',
    neutral: 'bg-muted-foreground',
    muted: 'bg-muted-foreground/70'
  }
  return tones[props.tone]
})
</script>

<template>
  <span :class="cn('inline-flex max-w-full min-w-0 items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium leading-5', toneClass, props.class)">
    <span :class="cn('h-1.5 w-1.5 shrink-0 rounded-full', dotClass, pulse && 'animate-pulse')" aria-hidden="true" />
    <span class="truncate"><slot /></span>
  </span>
</template>
