<script setup lang="ts">
import { AlertCircle, CheckCircle2, Info, TriangleAlert } from '@lucide/vue'
import { cn } from '~/lib/utils'

type NoticeTone = 'info' | 'success' | 'warning' | 'danger' | 'neutral'

const props = withDefaults(defineProps<{
  tone?: NoticeTone
  title?: string
  description?: string
  class?: string
}>(), {
  tone: 'info'
})

const toneClasses: Record<NoticeTone, string> = {
  info: 'state-info',
  success: 'state-success',
  warning: 'state-warning',
  danger: 'state-danger',
  neutral: 'border-border bg-surface-muted text-foreground'
}

const icons = {
  info: Info,
  success: CheckCircle2,
  warning: TriangleAlert,
  danger: AlertCircle,
  neutral: Info
}
</script>

<template>
  <aside
    :class="cn('flex items-start gap-3 border-l-4 px-4 py-3 text-sm leading-6', toneClasses[tone], props.class)"
    :role="tone === 'danger' ? 'alert' : 'status'"
    :aria-live="tone === 'danger' ? 'assertive' : 'polite'"
  >
    <component :is="icons[tone]" class="mt-1 h-4 w-4 shrink-0" aria-hidden="true" />
    <div class="min-w-0 flex-1">
      <p v-if="title" class="font-semibold">{{ title }}</p>
      <p v-if="description" :class="cn(title && 'mt-0.5', 'opacity-90')">{{ description }}</p>
      <slot />
    </div>
    <div v-if="$slots.actions" class="shrink-0"><slot name="actions" /></div>
  </aside>
</template>
