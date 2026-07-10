<script setup lang="ts">
import { Inbox } from '@lucide/vue'
import { cn } from '~/lib/utils'

const { t } = useI18n()
const props = defineProps<{
  title?: string
  description?: string
  class?: string
}>()
</script>

<template>
  <section :class="cn('flex min-h-40 flex-col items-start justify-center border border-dashed border-border bg-surface-muted px-5 py-8', props.class)" aria-live="polite">
    <slot name="icon">
      <div class="flex h-10 w-10 items-center justify-center border border-border bg-surface text-muted-foreground">
        <Inbox class="h-5 w-5" aria-hidden="true" />
      </div>
    </slot>
    <h2 class="mt-4 text-sm font-semibold text-foreground">{{ title || t('ui.states.emptyTitle') }}</h2>
    <p class="mt-1 max-w-xl text-sm leading-6 text-muted-foreground">{{ description || t('ui.states.emptyDescription') }}</p>
    <div v-if="$slots.actions" class="mt-4 flex flex-wrap gap-2">
      <slot name="actions" />
    </div>
  </section>
</template>
