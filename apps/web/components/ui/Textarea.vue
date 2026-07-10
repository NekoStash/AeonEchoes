<script setup lang="ts">
import { useAttrs } from 'vue'
import { cn } from '~/lib/utils'

defineOptions({ inheritAttrs: false })
const attrs = useAttrs()

const props = withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  disabled?: boolean
  invalid?: boolean
  rows?: number
  class?: string
}>(), {
  modelValue: '',
  disabled: false,
  invalid: false,
  rows: 5
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <textarea
    v-bind="attrs"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    :rows="rows"
    :aria-invalid="invalid || attrs['aria-invalid'] === 'true' ? 'true' : undefined"
    :class="cn(
      'focus-ring min-h-24 w-full min-w-0 resize-y border border-input bg-background px-3 py-2.5 text-sm leading-7 text-foreground transition-colors placeholder:text-muted-foreground hover:border-foreground/40 disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground disabled:opacity-70',
      invalid && 'border-state-danger focus-visible:ring-state-danger',
      props.class
    )"
    @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
  />
</template>
