<script setup lang="ts">
import { Inbox } from '@lucide/vue'
import { cn } from '~/lib/utils'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    title?: string
    description?: string
    iconLabel?: string
    class?: string
  }>(),
  {}
)
</script>

<template>
  <div :class="cn('surface-muted flex min-h-40 flex-col items-center justify-center rounded-2xl px-4 py-8 text-center', props.class)">
    <slot name="icon">
      <div class="flex h-11 w-11 items-center justify-center rounded-2xl bg-surface text-muted-foreground" :aria-label="iconLabel || undefined">
        <Inbox class="h-5 w-5" aria-hidden="true" />
      </div>
    </slot>
    <h3 class="mt-4 text-sm font-semibold text-foreground">{{ title || t('ui.states.emptyTitle') }}</h3>
    <p class="mt-1 max-w-md text-sm leading-6 text-muted-foreground">{{ description || t('ui.states.emptyDescription') }}</p>
    <div v-if="$slots.actions" class="mt-4 flex flex-wrap justify-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
