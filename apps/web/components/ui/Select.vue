<script setup lang="ts">
import { Check, ChevronDown, Search } from '@lucide/vue'
import { cn } from '~/lib/utils'

export interface SelectOption {
  label: string
  value: string
  description?: string
  disabled?: boolean
  disabledReason?: string
}

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    modelValue?: string
    options: SelectOption[]
    placeholder?: string
    disabled?: boolean
    searchable?: boolean
    searchPlaceholder?: string
    searchLabel?: string
    emptyText?: string
    class?: string
  }>(),
  {
    modelValue: '',
    placeholder: undefined,
    disabled: false,
    searchable: false,
    searchPlaceholder: undefined,
    searchLabel: undefined,
    emptyText: undefined
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const open = ref(false)
const searchQuery = ref('')
const root = ref<HTMLElement | null>(null)
const searchInput = ref<HTMLInputElement | null>(null)
const triggerId = `select-trigger-${Math.random().toString(36).slice(2, 10)}`
const listboxId = `select-listbox-${Math.random().toString(36).slice(2, 10)}`

const selectedOption = computed(() => props.options.find((option) => option.value === props.modelValue))
const displayLabel = computed(() => selectedOption.value?.label || props.placeholder || '')
const emptyResultText = computed(() => props.emptyText || t('ui.select.noResults'))
const filteredOptions = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  if (!props.searchable || !query) return props.options
  return props.options.filter((option) => [option.label, option.description, option.value]
    .filter(Boolean)
    .some((value) => String(value).toLowerCase().includes(query)))
})

async function openMenu() {
  if (props.disabled) return
  open.value = true
  await nextTick()
  searchInput.value?.focus()
}

function closeMenu() {
  open.value = false
  searchQuery.value = ''
}

function toggleMenu() {
  if (props.disabled) return
  if (open.value) closeMenu()
  else openMenu()
}

function handleTriggerKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' || event.key === ' ' || event.key === 'ArrowDown') {
    event.preventDefault()
    openMenu()
    return
  }
  if (event.key === 'Escape') {
    event.preventDefault()
    closeMenu()
  }
}

function handleSearchKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  event.preventDefault()
  event.stopPropagation()
  closeMenu()
}

function optionId(value: string) {
  return `${listboxId}-option-${value.replace(/[^A-Za-z0-9_-]/g, '-')}`
}

function choose(option: SelectOption) {
  if (props.disabled || option.disabled) return
  emit('update:modelValue', option.value)
  closeMenu()
}

function handleDocumentClick(event: MouseEvent) {
  if (!root.value || root.value.contains(event.target as Node)) return
  closeMenu()
}

onMounted(() => {
  document.addEventListener('click', handleDocumentClick)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick)
})
</script>

<template>
  <div ref="root" :class="cn('relative w-full', props.class)">
    <button
      :id="triggerId"
      type="button"
      :disabled="disabled"
      :aria-expanded="open"
      :aria-controls="open ? listboxId : undefined"
      :aria-activedescendant="selectedOption ? optionId(selectedOption.value) : undefined"
      aria-haspopup="listbox"
      class="flex h-10 w-full items-center justify-between gap-3 rounded-xl border border-input bg-background px-3 py-2 text-left text-sm text-foreground shadow-sm transition-colors hover:border-primary/35 hover:bg-muted/45 focus:border-ring focus-ring disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground disabled:opacity-70 dark:bg-muted/30 dark:hover:border-primary/45"
      @click.stop="toggleMenu"
      @keydown="handleTriggerKeydown"
    >
      <span :class="cn('truncate', !selectedOption && 'text-muted-foreground')">{{ displayLabel }}</span>
      <ChevronDown :class="cn('h-4 w-4 shrink-0 text-muted-foreground transition-transform', open && 'rotate-180 text-foreground')" aria-hidden="true" />
    </button>

    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="translate-y-1 opacity-0"
      enter-to-class="translate-y-0 opacity-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="translate-y-0 opacity-100"
      leave-to-class="translate-y-1 opacity-0"
    >
      <div
        v-if="open"
        :id="listboxId"
        class="absolute left-0 right-0 z-50 mt-2 max-h-72 min-w-0 overflow-auto rounded-xl border border-border bg-popover p-1 shadow-2xl shadow-black/25 subtle-scrollbar sm:min-w-full"
        role="listbox"
        :aria-labelledby="triggerId"
        @keydown.esc.prevent.stop="closeMenu"
      >
        <div v-if="searchable" class="sticky top-0 z-10 bg-popover p-1">
          <label class="flex items-center gap-2 rounded-lg border border-input bg-background px-2 py-1.5 text-sm shadow-sm dark:bg-muted/30">
            <Search class="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
            <span class="sr-only">{{ searchLabel || t('ui.select.search') }}</span>
            <input
              ref="searchInput"
              v-model="searchQuery"
              type="search"
              role="searchbox"
              :placeholder="searchPlaceholder || t('ui.select.search')"
              class="min-w-0 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
              @click.stop
              @keydown="handleSearchKeydown"
            >
          </label>
        </div>
        <button
          v-if="placeholder"
          :id="`${listboxId}-option-placeholder`"
          type="button"
          role="option"
          :aria-selected="!modelValue"
          class="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted focus-ring"
          @click="choose({ label: placeholder, value: '' })"
        >
          <span>{{ placeholder }}</span>
          <Check v-if="!modelValue" class="h-4 w-4" aria-hidden="true" />
        </button>
        <div v-if="filteredOptions.length === 0" class="px-3 py-2 text-sm text-muted-foreground" role="status">
          {{ emptyResultText }}
        </div>
        <button
          v-for="option in filteredOptions"
          :id="optionId(option.value)"
          :key="option.value"
          type="button"
          role="option"
          :aria-selected="option.value === modelValue"
          :disabled="option.disabled"
          :class="
            cn(
              'flex w-full items-center justify-between gap-3 rounded-lg px-3 py-2 text-left text-sm transition-colors focus-ring disabled:cursor-not-allowed disabled:opacity-45',
              option.value === modelValue ? 'bg-primary/12 text-foreground' : 'text-muted-foreground hover:bg-muted hover:text-foreground'
            )
          "
          @click="choose(option)"
        >
          <span class="min-w-0">
            <span class="block truncate">{{ option.label }}</span>
            <span v-if="option.description" class="mt-0.5 block truncate text-[11px] text-muted-foreground">{{ option.description }}</span>
            <span v-if="option.disabled && option.disabledReason" class="mt-0.5 block truncate text-[11px] text-state-warning-foreground">{{ option.disabledReason }}</span>
          </span>
          <Check v-if="option.value === modelValue" class="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
        </button>
      </div>
    </Transition>
  </div>
</template>
