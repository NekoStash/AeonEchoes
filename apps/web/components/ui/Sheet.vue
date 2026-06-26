<script setup lang="ts">
import { X } from '@lucide/vue'
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    description?: string
    side?: 'right' | 'left'
    class?: string
  }>(),
  {
    description: '',
    side: 'right'
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
      <div v-if="open" class="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm" @click="emit('update:open', false)" />
    </Transition>
    <Transition
      enter-active-class="transition duration-200 ease-out"
      :enter-from-class="side === 'right' ? 'translate-x-full opacity-0' : '-translate-x-full opacity-0'"
      enter-to-class="translate-x-0 opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="translate-x-0 opacity-100"
      :leave-to-class="side === 'right' ? 'translate-x-full opacity-0' : '-translate-x-full opacity-0'"
    >
      <aside
        v-if="open"
        :class="
          cn(
            'fixed top-0 z-50 flex h-[100dvh] w-[min(94vw,360px)] max-w-full flex-col border-border bg-card text-card-foreground shadow-2xl shadow-slate-950/20 dark:shadow-black/35',
            side === 'right' ? 'right-0 border-l' : 'left-0 border-r',
            props.class
          )
        "
      >
        <div v-if="title || description" class="flex shrink-0 items-start justify-between gap-4 border-b border-border px-5 py-4">
          <div class="min-w-0 flex-1">
            <h2 v-if="title" class="break-words text-lg font-semibold text-foreground">{{ title }}</h2>
            <p v-if="description" class="mt-1 break-words text-sm text-muted-foreground">{{ description }}</p>
          </div>
          <button type="button" class="shrink-0 rounded-lg p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-ring" @click="emit('update:open', false)">
            <X class="h-4 w-4" />
          </button>
        </div>
        <div :class="cn('min-h-0 flex-1 overflow-y-auto subtle-scrollbar', title || description ? 'px-4 py-4 sm:px-5 sm:py-5' : 'px-4 py-4')">
          <div v-if="!title && !description" class="mb-3 flex justify-end">
            <button type="button" class="shrink-0 rounded-lg p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-ring" @click="emit('update:open', false)">
              <X class="h-4 w-4" />
            </button>
          </div>
          <slot />
        </div>
      </aside>
    </Transition>
  </Teleport>
</template>
