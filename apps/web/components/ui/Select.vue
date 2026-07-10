<script setup lang="ts">
import { Check, ChevronDown, Search } from '@lucide/vue'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useAttrs, useId, watch } from 'vue'
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
const searchInput = ref<HTMLInputElement | null>(null)
const triggerId = `select-trigger-${useId()}`
const listboxId = `select-listbox-${useId()}`
const activeIndex = ref(-1)
const openAbove = ref(false)

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
const selectableOptions = computed(() => filteredOptions.value.filter((option) => !option.disabled))
const activeOption = computed(() => selectableOptions.value[activeIndex.value])

function optionId(value: string) {
  return `${listboxId}-option-${value.replace(/[^A-Za-z0-9_-]/g, '-') || 'empty'}`
}

function resetActiveIndex() {
  const selectedIndex = selectableOptions.value.findIndex((option) => option.value === props.modelValue)
  activeIndex.value = selectedIndex >= 0 ? selectedIndex : (selectableOptions.value.length > 0 ? 0 : -1)
}

async function openMenu() {
  if (props.disabled) return
  const triggerRect = trigger.value?.getBoundingClientRect()
  openAbove.value = Boolean(triggerRect && window.innerHeight - triggerRect.bottom < 300 && triggerRect.top > 300)
  open.value = true
  resetActiveIndex()
  await nextTick()
  if (props.searchable) searchInput.value?.focus()
  else root.value?.querySelector<HTMLElement>(`#${CSS.escape(listboxId)}`)?.focus()
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
  if (!root.value?.contains(event.target as Node)) closeMenu()
}

watch(filteredOptions, resetActiveIndex)
onMounted(() => document.addEventListener('click', handleDocumentClick))
onBeforeUnmount(() => document.removeEventListener('click', handleDocumentClick))
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
        'focus-ring flex h-10 w-full items-center justify-between gap-3 rounded-md border border-input bg-background px-3 py-2 text-left text-sm text-foreground transition-colors hover:border-foreground/40 disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground disabled:opacity-70',
        invalid && 'border-state-danger focus-visible:ring-state-danger'
      )"
      @click.stop="toggleMenu"
      @keydown="handleTriggerKeydown"
    >
      <span :class="cn('truncate', !selectedOption && 'text-muted-foreground')">{{ displayLabel }}</span>
      <ChevronDown :class="cn('h-4 w-4 shrink-0 text-muted-foreground transition-transform', open && 'rotate-180')" aria-hidden="true" />
    </button>

    <div
      v-if="open"
      :id="listboxId"
      :class="cn('absolute left-0 right-0 z-50 max-h-72 min-w-0 overflow-auto border border-border bg-popover p-1 subtle-scrollbar', openAbove ? 'bottom-full mb-1' : 'top-full mt-1')"
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
        type="button"
        role="option"
        :aria-selected="!modelValue"
        class="focus-ring flex w-full items-center justify-between px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted"
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
  </div>
</template>
