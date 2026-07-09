<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    title?: string
    description?: string
    class?: string
    bodyClass?: string
  }>(),
  {
    title: undefined,
    description: undefined
  }
)
</script>

<template>
  <section :class="cn('surface-panel min-w-0 rounded-2xl', props.class)">
    <DsPanelHeader v-if="title || description || $slots.header || $slots.actions" :title="title" :description="description">
      <slot name="header" />
      <template v-if="$slots.actions" #actions>
        <slot name="actions" />
      </template>
    </DsPanelHeader>
    <div :class="cn('space-y-4 p-4 sm:p-5', props.bodyClass)">
      <slot />
    </div>
  </section>
</template>
