<script setup lang="ts">
import { Search, X } from '@lucide/vue'
import { cn } from '~/lib/utils'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    modelValue?: string
    label?: string
    placeholder?: string
    clearLabel?: string
    disabled?: boolean
    class?: string
  }>(),
  {
    modelValue: '',
    label: undefined,
    placeholder: undefined,
    clearLabel: undefined,
    disabled: false
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  clear: []
}>()

function clearValue() {
  emit('update:modelValue', '')
  emit('clear')
}
</script>

<template>
  <label :class="cn('relative block min-w-0', props.class)">
    <span class="sr-only">{{ label || t('ui.search.label') }}</span>
    <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
    <input
      type="search"
      :value="modelValue"
      :placeholder="placeholder || t('ui.search.placeholder')"
      :disabled="disabled"
      class="h-10 w-full min-w-0 rounded-xl border border-input bg-background py-2 pl-9 pr-9 text-sm text-foreground shadow-sm transition-colors placeholder:text-muted-foreground hover:border-primary/35 focus:border-ring focus-ring disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground disabled:opacity-70 dark:bg-muted/30"
      @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    >
    <button
      v-if="modelValue"
      type="button"
      class="focus-ring absolute right-1.5 top-1/2 flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      :aria-label="clearLabel || t('ui.search.clear')"
      :disabled="disabled"
      @click="clearValue"
    >
      <X class="h-4 w-4" aria-hidden="true" />
    </button>
  </label>
</template>
