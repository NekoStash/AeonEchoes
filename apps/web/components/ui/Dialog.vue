<script setup lang="ts">
import { X } from '@lucide/vue'
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    description?: string
    class?: string
  }>(),
  {
    description: ''
  }
)

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div v-if="open" class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm" @click="emit('update:open', false)" />
    </Transition>
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="translate-y-3 scale-95 opacity-0"
      enter-to-class="translate-y-0 scale-100 opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="translate-y-0 scale-100 opacity-100"
      leave-to-class="translate-y-3 scale-95 opacity-0"
    >
      <section
        v-if="open"
        :class="cn('fixed left-1/2 top-1/2 z-50 w-[min(92vw,640px)] -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-card p-6 shadow-2xl', props.class)"
        role="dialog"
        aria-modal="true"
      >
        <div class="flex items-start justify-between gap-4">
          <div>
            <h2 class="text-lg font-semibold text-foreground">{{ title }}</h2>
            <p v-if="description" class="mt-1 text-sm text-muted-foreground">{{ description }}</p>
          </div>
          <button type="button" class="focus-ring rounded-lg p-2 text-muted-foreground hover:bg-muted hover:text-foreground" @click="emit('update:open', false)">
            <X class="h-4 w-4" />
          </button>
        </div>
        <div class="mt-5">
          <slot />
        </div>
      </section>
    </Transition>
  </Teleport>
</template>
