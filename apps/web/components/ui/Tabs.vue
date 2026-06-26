<script setup lang="ts">
import { cn } from '~/lib/utils'

export interface TabItem {
  label: string
  value: string
  badge?: string
}

const props = defineProps<{
  modelValue: string
  tabs: TabItem[]
  class?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <div :class="cn('flex max-w-full min-w-0 overflow-x-auto rounded-xl border border-border bg-muted/45 p-1 subtle-scrollbar', props.class)">
    <button
      v-for="tab in tabs"
      :key="tab.value"
      type="button"
      :class="
        cn(
          'focus-ring flex min-w-max shrink-0 items-center justify-center rounded-lg px-3 py-1.5 text-sm font-medium text-muted-foreground transition-all hover:text-foreground',
          modelValue === tab.value && 'bg-background/80 text-foreground shadow-sm'
        )
      "
      @click="emit('update:modelValue', tab.value)"
    >
      <span class="truncate">{{ tab.label }}</span>
      <span v-if="tab.badge" class="ml-1 rounded-full bg-primary/15 px-1.5 py-0.5 text-[10px] text-primary">
        {{ tab.badge }}
      </span>
    </button>
  </div>
</template>
