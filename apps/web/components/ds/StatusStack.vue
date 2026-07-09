<script setup lang="ts">
import { cn } from '~/lib/utils'

export interface StatusStackItem {
  id: string
  tone?: 'info' | 'success' | 'warning' | 'danger' | 'neutral'
  title?: string
  description?: string
}

const props = withDefaults(
  defineProps<{
    items?: StatusStackItem[]
    class?: string
  }>(),
  {
    items: () => []
  }
)
</script>

<template>
  <div :class="cn('space-y-2', props.class)">
    <slot>
      <UiAlert
        v-for="item in items"
        :key="item.id"
        :tone="item.tone || 'info'"
        :title="item.title"
        :description="item.description"
      />
    </slot>
  </div>
</template>
