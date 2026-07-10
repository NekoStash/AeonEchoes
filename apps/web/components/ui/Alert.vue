<script setup lang="ts">
import { AlertCircle, CheckCircle2, Info, TriangleAlert } from '@lucide/vue'
import { cn } from '~/lib/utils'

type AlertTone = 'info' | 'success' | 'warning' | 'danger' | 'neutral'

const props = withDefaults(defineProps<{
  tone?: AlertTone
  title?: string
  description?: string
  class?: string
}>(), {
  tone: 'info',
  title: undefined,
  description: undefined
})

const toneClass = computed(() => ({
  info: 'state-info',
  success: 'state-success',
  warning: 'state-warning',
  danger: 'state-danger',
  neutral: 'border-border bg-surface-muted text-foreground'
})[props.tone])

const iconByTone = {
  info: Info,
  success: CheckCircle2,
  warning: TriangleAlert,
  danger: AlertCircle,
  neutral: Info
}
</script>

<template>
  <div :class="cn('flex items-start gap-3 rounded-md border px-4 py-3 text-sm leading-6', toneClass, props.class)" :role="tone === 'danger' ? 'alert' : 'status'">
    <slot name="icon"><component :is="iconByTone[tone]" class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" /></slot>
    <div class="min-w-0 flex-1">
      <p v-if="title" class="font-semibold">{{ title }}</p>
      <p v-if="description" :class="cn(title && 'mt-1', 'opacity-90')">{{ description }}</p>
      <slot />
    </div>
    <div v-if="$slots.actions" class="shrink-0"><slot name="actions" /></div>
  </div>
</template>
