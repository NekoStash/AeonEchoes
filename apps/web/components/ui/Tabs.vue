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
  <div
    role="tablist"
    :class="cn('flex max-w-full min-w-0 gap-1 overflow-x-auto rounded-2xl border border-border bg-muted/35 p-1 subtle-scrollbar', props.class)"
  >
    <button
      v-for="tab in tabs"
      :key="tab.value"
      type="button"
      role="tab"
      :aria-selected="modelValue === tab.value"
      :class="
        cn(
          'focus-ring flex min-w-max shrink-0 items-center justify-center rounded-xl px-3 py-2 text-sm font-medium leading-5 text-muted-foreground transition-all hover:bg-background/60 hover:text-foreground',
          modelValue === tab.value && 'bg-background text-foreground shadow-sm ring-1 ring-border/70'
        )
      "
      @click="emit('update:modelValue', tab.value)"
    >
      <span class="truncate">{{ tab.label }}</span>
      <span v-if="tab.badge" class="ml-1.5 rounded-full bg-primary/15 px-1.5 py-0.5 text-[10px] leading-none text-primary">
        {{ tab.badge }}
      </span>
    </button>
  </div>
</template>
