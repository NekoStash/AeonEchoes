<script setup lang="ts">
import { AlertCircle, CheckCircle2, Info, TriangleAlert, X } from '@lucide/vue'
import { cn } from '~/lib/utils'
import type { ToastMessage } from '~/shared/composables/useToast'

const { t } = useI18n()

const props = defineProps<{
  message: ToastMessage
}>()

const emit = defineEmits<{
  dismiss: [id: number]
}>()

const toneClasses = {
  info: 'border-state-info-border bg-state-info-surface text-state-info-foreground',
  success: 'border-state-success-border bg-state-success-surface text-state-success-foreground',
  warning: 'border-state-warning-border bg-state-warning-surface text-state-warning-foreground',
  danger: 'border-state-danger-border bg-state-danger-surface text-state-danger-foreground'
}
const icons = {
  info: Info,
  success: CheckCircle2,
  warning: TriangleAlert,
  danger: AlertCircle
}
</script>

<template>
  <article
    :class="cn('pointer-events-auto flex w-full items-start gap-3 border border-l-4 px-4 py-3', toneClasses[message.tone])"
    :role="message.tone === 'danger' ? 'alert' : 'status'"
    :aria-live="message.tone === 'danger' ? 'assertive' : 'polite'"
  >
    <component :is="icons[message.tone]" class="mt-1 h-4 w-4 shrink-0" aria-hidden="true" />
    <div class="min-w-0 flex-1">
      <p class="text-sm font-semibold">{{ message.title }}</p>
      <p v-if="message.description" class="mt-0.5 text-sm leading-6 opacity-90">{{ message.description }}</p>
    </div>
    <button type="button" class="focus-ring -mr-1 flex h-8 w-8 shrink-0 items-center justify-center hover:bg-black/10" :aria-label="t('actions.dismiss')" @click="emit('dismiss', message.id)">
      <X class="h-4 w-4" aria-hidden="true" />
    </button>
  </article>
</template>
