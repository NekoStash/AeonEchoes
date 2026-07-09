<script setup lang="ts">
import { LoaderCircle } from '@lucide/vue'
import { cn } from '~/lib/utils'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    title?: string
    description?: string
    class?: string
  }>(),
  {}
)
</script>

<template>
  <div :class="cn('surface-muted flex min-h-40 flex-col items-center justify-center rounded-2xl px-4 py-8 text-center', props.class)" role="status" aria-live="polite">
    <LoaderCircle class="h-6 w-6 animate-spin text-primary" aria-hidden="true" />
    <h3 class="mt-4 text-sm font-semibold text-foreground">{{ title || t('ui.states.loadingTitle') }}</h3>
    <p class="mt-1 max-w-md text-sm leading-6 text-muted-foreground">{{ description || t('ui.states.loadingDescription') }}</p>
  </div>
</template>
