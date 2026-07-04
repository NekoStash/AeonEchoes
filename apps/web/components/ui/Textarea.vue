<script setup lang="ts">
import { useAttrs } from 'vue'
import { cn } from '~/lib/utils'

defineOptions({ inheritAttrs: false })

const attrs = useAttrs()

const props = withDefaults(
  defineProps<{
    modelValue?: string
    placeholder?: string
    disabled?: boolean
    rows?: number
    class?: string
  }>(),
  {
    modelValue: '',
    rows: 5
  }
)

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
    :class="
      cn(
        'min-h-24 w-full min-w-0 resize-y rounded-xl border border-input bg-background px-3.5 py-3 text-sm leading-7 text-foreground shadow-sm transition-colors placeholder:text-muted-foreground hover:border-primary/35 focus:border-ring focus-ring disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground disabled:opacity-70 dark:bg-muted/30 dark:hover:border-primary/45',
        props.class
      )
    "
    @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
  />
</template>
