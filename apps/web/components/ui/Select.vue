<script setup lang="ts">
import { Check, ChevronDown } from '@lucide/vue'
import { cn } from '~/lib/utils'

export interface SelectOption {
  label: string
  value: string
  description?: string
  disabled?: boolean
}

const props = defineProps<{
  modelValue?: string
  options: SelectOption[]
  placeholder?: string
  disabled?: boolean
  class?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)

const selectedOption = computed(() => props.options.find((option) => option.value === props.modelValue))
const displayLabel = computed(() => selectedOption.value?.label || props.placeholder || '')

function choose(option: SelectOption) {
  if (props.disabled || option.disabled) return
  emit('update:modelValue', option.value)
  open.value = false
}

function handleDocumentClick(event: MouseEvent) {
  if (!root.value || root.value.contains(event.target as Node)) return
  open.value = false
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
      type="button"
      :disabled="disabled"
      :aria-expanded="open"
      class="flex h-10 w-full items-center justify-between gap-3 rounded-xl border border-input bg-background px-3 py-2 text-left text-sm text-foreground shadow-sm transition-colors hover:border-primary/35 hover:bg-muted/45 focus:border-ring focus-ring disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground disabled:opacity-70 dark:bg-muted/30 dark:hover:border-primary/45"
      @click.stop="open = !open"
    >
      <span :class="cn('truncate', !selectedOption && 'text-muted-foreground')">{{ displayLabel }}</span>
      <ChevronDown :class="cn('h-4 w-4 shrink-0 text-muted-foreground transition-transform', open && 'rotate-180 text-foreground')" />
    </button>

    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="translate-y-1 opacity-0"
      enter-to-class="translate-y-0 opacity-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="translate-y-0 opacity-100"
      leave-to-class="translate-y-1 opacity-0"
    >
      <div v-if="open" class="absolute left-0 right-0 z-50 mt-2 max-h-72 min-w-0 overflow-auto rounded-xl border border-border bg-popover p-1 shadow-2xl shadow-black/25 subtle-scrollbar sm:min-w-full">
        <button
          v-if="placeholder"
          type="button"
          class="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted focus-ring"
          @click="choose({ label: placeholder, value: '' })"
        >
          <span>{{ placeholder }}</span>
          <Check v-if="!modelValue" class="h-4 w-4" />
        </button>
        <button
          v-for="option in options"
          :key="option.value"
          type="button"
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
          </span>
          <Check v-if="option.value === modelValue" class="h-4 w-4 shrink-0 text-primary" />
        </button>
      </div>
    </Transition>
  </div>
</template>
