<script setup lang="ts">
import { LoaderCircle } from '@lucide/vue'
import { cn } from '~/lib/utils'

type AsyncStatus = 'idle' | 'loading' | 'error' | 'empty' | 'ready'

const { t } = useI18n()
const props = withDefaults(defineProps<{
  status: AsyncStatus
  error?: unknown
  loadingTitle?: string
  loadingDescription?: string
  errorTitle?: string
  errorDescription?: string
  emptyTitle?: string
  emptyDescription?: string
  class?: string
}>(), {
  error: undefined
})

watch(
  () => [props.status, props.error] as const,
  ([status, error]) => {
    if (status !== 'error') return
    console.error('[AeonEchoes UI] Async state failed.', error || props.errorDescription || props.errorTitle || 'Unknown asynchronous error')
  },
  { immediate: true }
)
</script>

<template>
  <div :class="cn('min-w-0', props.class)">
    <slot v-if="status === 'ready'" />
    <slot v-else-if="status === 'idle'" name="idle" />
    <slot v-else-if="status === 'loading'" name="loading">
      <div class="flex min-h-40 flex-col items-start justify-center border border-border bg-surface-muted px-5 py-8" role="status" aria-live="polite">
        <LoaderCircle class="h-5 w-5 animate-spin text-foreground" aria-hidden="true" />
        <p class="mt-4 text-sm font-semibold text-foreground">{{ loadingTitle || t('ui.states.loadingTitle') }}</p>
        <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ loadingDescription || t('ui.states.loadingDescription') }}</p>
      </div>
    </slot>
    <slot v-else-if="status === 'error'" name="error" :error="error">
      <UiInlineNotice tone="danger" :title="errorTitle || t('ui.states.errorTitle')" :description="errorDescription || t('ui.states.errorDescription')" />
    </slot>
    <slot v-else name="empty">
      <UiEmptyState :title="emptyTitle" :description="emptyDescription" />
    </slot>
  </div>
</template>
