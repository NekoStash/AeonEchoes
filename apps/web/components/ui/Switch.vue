<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    modelValue?: boolean
    disabled?: boolean
    label?: string
    description?: string
    class?: string
  }>(),
  {
    modelValue: false,
    disabled: false
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

function toggle() {
  if (props.disabled) return
  emit('update:modelValue', !props.modelValue)
}
</script>

<template>
  <button
    type="button"
    role="switch"
    :aria-checked="modelValue"
    :disabled="disabled"
    :class="
      cn(
        'group flex min-h-14 w-full items-center justify-between gap-4 rounded-2xl border border-border bg-card/75 px-4 py-3 text-left text-sm leading-6 shadow-sm transition-all hover:border-primary/30 hover:bg-muted/50 focus-ring disabled:cursor-not-allowed disabled:opacity-50',
        modelValue && 'border-primary/35 bg-primary/10',
        props.class
      )
    "
    @click="toggle"
  >
    <span class="min-w-0 flex-1">
      <span v-if="label" class="block font-medium leading-5 text-foreground">{{ label }}</span>
      <span v-if="description" class="mt-1 block text-xs leading-5 text-muted-foreground">{{ description }}</span>
      <slot v-if="!label" />
    </span>
    <span
      :class="
        cn(
          'relative inline-flex h-7 w-12 shrink-0 items-center rounded-full border border-border bg-muted p-0.5 transition-colors duration-200 ease-out',
          modelValue && 'border-primary bg-primary'
        )
      "
    >
      <span
        :class="
          cn(
            'block h-6 w-6 rounded-full bg-background shadow-sm ring-1 ring-black/5 transition-transform duration-200 ease-out',
            modelValue && 'translate-x-5 bg-primary-foreground'
          )
        "
      />
    </span>
  </button>
</template>
