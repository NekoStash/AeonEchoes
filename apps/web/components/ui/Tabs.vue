<script setup lang="ts">
import { cn } from '~/lib/utils'

export interface TabItem {
  label: string
  value: string
  badge?: string
  disabled?: boolean
}

const props = withDefaults(
  defineProps<{
    modelValue: string
    tabs: TabItem[]
    orientation?: 'horizontal' | 'vertical'
    panelIdPrefix?: string
    class?: string
  }>(),
  {
    orientation: 'horizontal',
    panelIdPrefix: undefined
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const root = ref<HTMLElement | null>(null)
const fallbackId = `tabs-${Math.random().toString(36).slice(2, 10)}`
const idPrefix = computed(() => props.panelIdPrefix || fallbackId)

function tabId(value: string) {
  return `${idPrefix.value}-tab-${value}`
}

function panelId(value: string) {
  return `${idPrefix.value}-panel-${value}`
}

function selectableTabs() {
  return props.tabs.filter((tab) => !tab.disabled)
}

async function activateTab(value: string) {
  emit('update:modelValue', value)
  await nextTick()
  root.value?.querySelector<HTMLElement>(`[data-tab-value="${CSS.escape(value)}"]`)?.focus()
}

function handleKeydown(event: KeyboardEvent, currentValue: string) {
  const horizontalKeys = ['ArrowLeft', 'ArrowRight']
  const verticalKeys = ['ArrowUp', 'ArrowDown']
  const handledKeys = props.orientation === 'vertical' ? verticalKeys : horizontalKeys
  if (![...handledKeys, 'Home', 'End'].includes(event.key)) return

  const tabs = selectableTabs()
  if (tabs.length === 0) return

  event.preventDefault()
  const currentIndex = Math.max(0, tabs.findIndex((tab) => tab.value === currentValue))
  let nextIndex = currentIndex

  if (event.key === 'Home') nextIndex = 0
  else if (event.key === 'End') nextIndex = tabs.length - 1
  else if (event.key === 'ArrowRight' || event.key === 'ArrowDown') nextIndex = (currentIndex + 1) % tabs.length
  else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') nextIndex = (currentIndex - 1 + tabs.length) % tabs.length

  const nextTab = tabs[nextIndex]
  if (nextTab) activateTab(nextTab.value)
}
</script>

<template>
  <div
    ref="root"
    role="tablist"
    :aria-orientation="orientation"
    :class="cn('flex max-w-full min-w-0 gap-1 overflow-x-auto rounded-2xl border border-border bg-muted/35 p-1 subtle-scrollbar', orientation === 'vertical' && 'flex-col overflow-x-visible', props.class)"
  >
    <button
      v-for="tab in tabs"
      :id="tabId(tab.value)"
      :key="tab.value"
      type="button"
      role="tab"
      :data-tab-value="tab.value"
      :disabled="tab.disabled"
      :aria-selected="modelValue === tab.value"
      :aria-controls="panelId(tab.value)"
      :tabindex="modelValue === tab.value ? 0 : -1"
      :class="
        cn(
          'focus-ring flex min-w-max shrink-0 items-center justify-center rounded-xl px-3 py-2 text-sm font-medium leading-5 text-muted-foreground transition-all hover:bg-background/60 hover:text-foreground disabled:cursor-not-allowed disabled:opacity-45',
          orientation === 'vertical' && 'w-full justify-start',
          modelValue === tab.value && 'bg-background text-foreground shadow-sm ring-1 ring-border/70'
        )
      "
      @click="!tab.disabled && emit('update:modelValue', tab.value)"
      @keydown="handleKeydown($event, tab.value)"
    >
      <span class="truncate">{{ tab.label }}</span>
      <span v-if="tab.badge" class="ml-1.5 rounded-full bg-primary/15 px-1.5 py-0.5 text-[10px] leading-none text-primary">
        {{ tab.badge }}
      </span>
    </button>
  </div>
</template>
