<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = withDefaults(defineProps<{
  modelValue?: boolean
  disabled?: boolean
  label?: string
  description?: string
  class?: string
}>(), {
  modelValue: false,
  disabled: false
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

function toggle() {
  if (!props.disabled) emit('update:modelValue', !props.modelValue)
}
</script>

<template>
  <button
    type="button"
    role="switch"
    :aria-checked="modelValue"
    :disabled="disabled"
    :class="cn('focus-ring group flex min-h-14 w-full items-center justify-between gap-4 border border-border bg-card px-4 py-3 text-left text-sm leading-6 transition-colors hover:border-foreground/35 hover:bg-muted disabled:cursor-not-allowed disabled:opacity-50', modelValue && 'border-foreground/45 bg-accent', props.class)"
    @click="toggle"
  >
    <span class="min-w-0 flex-1">
      <span v-if="label" class="block font-semibold leading-5 text-foreground">{{ label }}</span>
      <span v-if="description" class="mt-1 block text-xs leading-5 text-muted-foreground">{{ description }}</span>
      <slot v-if="!label" />
    </span>
    <span data-aeon-square :class="cn('relative inline-flex h-6 w-11 shrink-0 items-center border border-border bg-muted p-0.5 transition-colors', modelValue && 'border-foreground bg-foreground')">
      <span data-aeon-square :class="cn('block h-5 w-5 bg-background transition-transform', modelValue && 'translate-x-5')" />
    </span>
  </button>
</template>
