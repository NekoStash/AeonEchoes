<script setup lang="ts">
import { AlertTriangle } from '@lucide/vue'
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
  <div :class="cn('state-danger flex min-h-40 flex-col items-center justify-center rounded-2xl border px-4 py-8 text-center', props.class)" role="alert">
    <AlertTriangle class="h-6 w-6" aria-hidden="true" />
    <h3 class="mt-4 text-sm font-semibold">{{ title || t('ui.states.errorTitle') }}</h3>
    <p class="mt-1 max-w-md text-sm leading-6 opacity-85">{{ description || t('ui.states.errorDescription') }}</p>
    <div v-if="$slots.actions" class="mt-4 flex flex-wrap justify-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
