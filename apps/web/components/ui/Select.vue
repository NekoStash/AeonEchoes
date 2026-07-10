<script setup lang="ts">
import { Check, ChevronDown, Search } from '@lucide/vue'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useAttrs, useId, watch, type CSSProperties } from 'vue'
import { cn } from '~/lib/utils'

export interface SelectOption {
  label: string
  value: string
  description?: string
  disabled?: boolean
  disabledReason?: string
}

defineOptions({ inheritAttrs: false })
const attrs = useAttrs()
const { t } = useI18n()
const props = withDefaults(defineProps<{
  id?: string
  modelValue?: string
  options: SelectOption[]
  placeholder?: string
  disabled?: boolean
  searchable?: boolean
  searchPlaceholder?: string
  searchLabel?: string
  emptyText?: string
  invalid?: boolean
  ariaLabel?: string
  class?: string
}>(), {
  id: undefined,
  modelValue: '',
  placeholder: undefined,
  disabled: false,
  searchable: false,
  searchPlaceholder: undefined,
  searchLabel: undefined,
  emptyText: undefined,
  invalid: false,
  ariaLabel: undefined
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const open = ref(false)
const searchQuery = ref('')
const root = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const listbox = ref<HTMLElement | null>(null)
const searchInput = ref<HTMLInputElement | null>(null)
const generatedTriggerId = `select-trigger-${useId()}`
const triggerId = computed(() => props.id || generatedTriggerId)
const listboxId = `select-listbox-${useId()}`
const activeIndex = ref(-1)
const menuStyle = ref<CSSProperties>({})

const selectedOption = computed(() => props.options.find((option) => option.value === props.modelValue))
const displayLabel = computed(() => selectedOption.value?.label || props.placeholder || '')
const emptyResultText = computed(() => props.emptyText || t('ui.select.noResults'))
const resolvedAriaLabel = computed(() => props.ariaLabel || (attrs['aria-label'] as string | undefined))
const filteredOptions = computed(() => {
  const query = searchQuery.value.trim().toLocaleLowerCase()
  if (!props.searchable || !query) return props.options
  return props.options.filter((option) => [option.label, option.description, option.value]
    .filter(Boolean)
    .some((value) => String(value).toLocaleLowerCase().includes(query)))
})
const placeholderOption = computed<SelectOption | undefined>(() => props.placeholder ? { label: props.placeholder, value: '' } : undefined)
const selectableOptions = computed(() => [
  ...(placeholderOption.value ? [placeholderOption.value] : []),
  ...filteredOptions.value
].filter((option) => !option.disabled))
const activeOption = computed(() => selectableOptions.value[activeIndex.value])

function optionId(value: string) {
  return `${listboxId}-option-${value.replace(/[^A-Za-z0-9_-]/g, '-') || 'empty'}`
}

function resetActiveIndex() {
  const selectedIndex = selectableOptions.value.findIndex((option) => option.value === props.modelValue)
  activeIndex.value = selectedIndex >= 0 ? selectedIndex : (selectableOptions.value.length > 0 ? 0 : -1)
}

function updateMenuPosition() {
  const triggerRect = trigger.value?.getBoundingClientRect()
  if (!triggerRect) {
    console.error('[AeonEchoes UI] Select menu opened without a mounted trigger.')
    return
  }
  const viewportPadding = 8
  const gap = 4
  const spaceBelow = window.innerHeight - triggerRect.bottom - viewportPadding - gap
  const spaceAbove = triggerRect.top - viewportPadding - gap
  const openAbove = spaceBelow < 240 && spaceAbove > spaceBelow
  const availableHeight = Math.max(120, Math.min(288, openAbove ? spaceAbove : spaceBelow))
  menuStyle.value = {
    left: `${Math.max(viewportPadding, triggerRect.left)}px`,
    width: `${Math.min(triggerRect.width, window.innerWidth - viewportPadding * 2)}px`,
    maxHeight: `${availableHeight}px`,
    top: openAbove ? undefined : `${triggerRect.bottom + gap}px`,
    bottom: openAbove ? `${window.innerHeight - triggerRect.top + gap}px` : undefined
  }
}

async function openMenu() {
  if (props.disabled) return
  updateMenuPosition()
  open.value = true
  resetActiveIndex()
  await nextTick()
  if (props.searchable) searchInput.value?.focus()
  else listbox.value?.focus()
}

function closeMenu(restoreFocus = false) {
  open.value = false
  searchQuery.value = ''
  activeIndex.value = -1
  if (restoreFocus) nextTick(() => trigger.value?.focus())
}

function toggleMenu() {
  if (open.value) closeMenu()
  else openMenu()
}

function choose(option: SelectOption) {
  if (props.disabled || option.disabled) return
  emit('update:modelValue', option.value)
  closeMenu(true)
}

function moveActive(step: number) {
  if (selectableOptions.value.length === 0) return
  activeIndex.value = (activeIndex.value + step + selectableOptions.value.length) % selectableOptions.value.length
  nextTick(() => document.getElementById(optionId(activeOption.value?.value || ''))?.scrollIntoView({ block: 'nearest' }))
}

function handleTriggerKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' || event.key === ' ' || event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    openMenu()
    return
  }
  if (event.key === 'Escape') closeMenu()
}

function handleListboxKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    closeMenu(true)
    return
  }
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    moveActive(1)
    return
  }
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    moveActive(-1)
    return
  }
  if (event.key === 'Home') {
    event.preventDefault()
    activeIndex.value = selectableOptions.value.length > 0 ? 0 : -1
    return
  }
  if (event.key === 'End') {
    event.preventDefault()
    activeIndex.value = selectableOptions.value.length - 1
    return
  }
  if ((event.key === 'Enter' || event.key === ' ') && activeOption.value) {
    event.preventDefault()
    choose(activeOption.value)
  }
}

function handleDocumentClick(event: MouseEvent) {
  const target = event.target as Node
  if (!root.value?.contains(target) && !listbox.value?.contains(target)) closeMenu()
}

function handleViewportScroll(event: Event) {
  if (!open.value || listbox.value?.contains(event.target as Node)) return
  closeMenu()
}

function handleViewportResize() {
  if (!open.value) return
  updateMenuPosition()
}

watch(filteredOptions, resetActiveIndex)
onMounted(() => {
  document.addEventListener('click', handleDocumentClick)
  document.addEventListener('scroll', handleViewportScroll, true)
  window.addEventListener('resize', handleViewportResize)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick)
  document.removeEventListener('scroll', handleViewportScroll, true)
  window.removeEventListener('resize', handleViewportResize)
})
</script>

<template>
  <div ref="root" :class="cn('relative w-full', props.class)">
    <button
      v-bind="attrs"
      :id="triggerId"
      ref="trigger"
      type="button"
      :disabled="disabled"
      :aria-label="resolvedAriaLabel"
      :aria-expanded="open"
      :aria-controls="open ? listboxId : undefined"
      :aria-activedescendant="open && activeOption ? optionId(activeOption.value) : undefined"
      :aria-invalid="invalid || attrs['aria-invalid'] === 'true' ? 'true' : undefined"
      aria-haspopup="listbox"
      :class="cn(
        'focus-ring flex h-10 w-full items-center justify-between gap-3 border border-input bg-background px-3 py-2 text-left text-sm text-foreground transition-colors hover:border-foreground/40 disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground disabled:opacity-70',
        invalid && 'border-state-danger focus-visible:ring-state-danger'
      )"
      @click.stop="toggleMenu"
      @keydown="handleTriggerKeydown"
    >
      <span :class="cn('truncate', !selectedOption && 'text-muted-foreground')">{{ displayLabel }}</span>
      <ChevronDown :class="cn('h-4 w-4 shrink-0 text-muted-foreground transition-transform', open && 'rotate-180')" aria-hidden="true" />
    </button>

    <Teleport to="body">
    <div
      v-if="open"
      :id="listboxId"
      ref="listbox"
      :style="menuStyle"
      class="fixed z-[70] min-w-0 overflow-auto border border-border bg-popover p-1 subtle-scrollbar"
      role="listbox"
      :aria-label="resolvedAriaLabel"
      :aria-labelledby="resolvedAriaLabel ? undefined : triggerId"
      tabindex="-1"
      @keydown="handleListboxKeydown"
    >
      <div v-if="searchable" class="sticky top-0 z-10 bg-popover p-1">
        <label class="flex items-center gap-2 border border-input bg-background px-2 py-1.5 text-sm">
          <Search class="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <span class="sr-only">{{ searchLabel || t('ui.select.search') }}</span>
          <input
            ref="searchInput"
            v-model="searchQuery"
            type="search"
            role="searchbox"
            :aria-label="searchLabel || t('ui.select.search')"
            :placeholder="searchPlaceholder || t('ui.select.search')"
            class="min-w-0 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
            @click.stop
            @keydown="handleListboxKeydown"
          >
        </label>
      </div>
      <button
        v-if="placeholder"
        :id="optionId('')"
        type="button"
        role="option"
        :aria-selected="!modelValue"
        :class="cn(
          'focus-ring flex w-full items-center justify-between px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted',
          activeOption?.value === '' && modelValue !== '' && 'bg-muted text-foreground'
        )"
        @mouseenter="activeIndex = selectableOptions.findIndex((item) => item.value === '')"
        @click="choose({ label: placeholder, value: '' })"
      >
        <span>{{ placeholder }}</span>
        <Check v-if="!modelValue" class="h-4 w-4" aria-hidden="true" />
      </button>
      <div v-if="filteredOptions.length === 0" class="px-3 py-3 text-sm text-muted-foreground" role="status">{{ emptyResultText }}</div>
      <button
        v-for="option in filteredOptions"
        :id="optionId(option.value)"
        :key="option.value"
        type="button"
        role="option"
        :aria-selected="option.value === modelValue"
        :disabled="option.disabled"
        :class="cn(
          'focus-ring flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-45',
          option.value === modelValue ? 'bg-foreground text-background' : 'text-muted-foreground hover:bg-muted hover:text-foreground',
          option.value === activeOption?.value && option.value !== modelValue && 'bg-muted text-foreground'
        )"
        @mouseenter="activeIndex = selectableOptions.findIndex((item) => item.value === option.value)"
        @click="choose(option)"
      >
        <span class="min-w-0">
          <span class="block truncate">{{ option.label }}</span>
          <span v-if="option.description" class="mt-0.5 block truncate text-[11px] opacity-75">{{ option.description }}</span>
          <span v-if="option.disabled && option.disabledReason" class="mt-0.5 block truncate text-[11px] text-state-warning-foreground">{{ option.disabledReason }}</span>
        </span>
        <Check v-if="option.value === modelValue" class="h-4 w-4 shrink-0" aria-hidden="true" />
      </button>
    </div>
    </Teleport>
  </div>
</template>
