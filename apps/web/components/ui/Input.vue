<script setup lang="ts">
import { useAttrs } from 'vue'
import { cn } from '~/lib/utils'

defineOptions({ inheritAttrs: false })

const attrs = useAttrs()

const props = withDefaults(
  defineProps<{
    modelValue?: string | number
    type?: string
    placeholder?: string
    disabled?: boolean
    class?: string
  }>(),
  {
    type: 'text',
    modelValue: ''
  }
)

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
    :class="
      cn(
        'h-11 w-full min-w-0 rounded-xl border border-input bg-background px-3.5 py-2 text-sm leading-6 text-foreground shadow-sm transition-colors placeholder:text-muted-foreground hover:border-primary/35 focus:border-ring focus-ring disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground disabled:opacity-70 dark:bg-muted/30 dark:hover:border-primary/45',
        props.class
      )
    "
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  />
</template>
