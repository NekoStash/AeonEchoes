<script setup lang="ts">
import { useAttrs } from 'vue'
import { cn } from '~/lib/utils'

defineOptions({ inheritAttrs: false })
const attrs = useAttrs()

const props = withDefaults(defineProps<{
  modelValue?: string | number
  type?: string
  placeholder?: string
  disabled?: boolean
  invalid?: boolean
  class?: string
}>(), {
  type: 'text',
  modelValue: '',
  disabled: false,
  invalid: false
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <input
    v-bind="attrs"
    :type="type"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    :aria-invalid="invalid || attrs['aria-invalid'] === 'true' ? 'true' : undefined"
    :class="cn(
      'focus-ring h-10 w-full min-w-0 border border-input bg-background px-3 py-2 text-sm leading-6 text-foreground transition-colors placeholder:text-muted-foreground hover:border-foreground/40 disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground disabled:opacity-70',
      invalid && 'border-state-danger focus-visible:ring-state-danger',
      props.class
    )"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  >
</template>
