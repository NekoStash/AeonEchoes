<script setup lang="ts">
import { cn } from '~/lib/utils'

type MetricTone = 'info' | 'success' | 'warning' | 'danger' | 'neutral' | 'cyan' | 'violet' | 'gold' | 'rose' | 'emerald'

const props = defineProps<{
  label: string
  value: string | number
  hint?: string
  tone?: MetricTone
}>()

const toneClass = computed(() => {
  const tones: Record<MetricTone, string> = {
    info: 'border-state-info-border bg-state-info-surface text-state-info-foreground',
    success: 'border-state-success-border bg-state-success-surface text-state-success-foreground',
    warning: 'border-state-warning-border bg-state-warning-surface text-state-warning-foreground',
    danger: 'border-state-danger-border bg-state-danger-surface text-state-danger-foreground',
    neutral: 'border-border bg-surface text-foreground',
    cyan: 'border-state-info-border bg-state-info-surface text-state-info-foreground',
    violet: 'border-state-info-border bg-state-info-surface text-state-info-foreground',
    gold: 'border-state-warning-border bg-state-warning-surface text-state-warning-foreground',
    rose: 'border-state-danger-border bg-state-danger-surface text-state-danger-foreground',
    emerald: 'border-state-success-border bg-state-success-surface text-state-success-foreground'
  }
  return tones[props.tone || 'info']
})
</script>

<template>
  <div :class="cn('rounded-2xl border p-4 shadow-sm', toneClass)">
    <p class="text-xs uppercase tracking-[0.22em] text-current/65">{{ label }}</p>
    <p class="mt-3 text-2xl font-semibold text-current">{{ value }}</p>
    <p v-if="hint" class="mt-1 text-xs text-current/65">{{ hint }}</p>
  </div>
</template>
