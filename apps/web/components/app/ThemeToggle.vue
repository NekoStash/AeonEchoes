<script setup lang="ts">
import { Check, Monitor, Moon, Sun } from '@lucide/vue'
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    compact?: boolean
  }>(),
  {
    compact: false
  }
)

const colorMode = useColorMode()
const { t } = useI18n()

const open = ref(false)
const root = ref<HTMLElement | null>(null)

const systemOption = { value: 'system', icon: Monitor, labelKey: 'theme.system' }
const options = [
  systemOption,
  { value: 'light', icon: Sun, labelKey: 'theme.light' },
  { value: 'dark', icon: Moon, labelKey: 'theme.dark' }
]

const activeOption = computed(() => options.find((option) => option.value === colorMode.preference) || systemOption)

function selectTheme(value: string) {
  colorMode.preference = value
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
  <div ref="root" class="relative inline-flex">
    <div v-if="!compact" class="inline-flex rounded-xl border border-border bg-card p-1 shadow-sm">
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        :title="t(option.labelKey)"
        :class="[
          'inline-flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:text-foreground focus-ring',
          colorMode.preference === option.value && 'bg-muted text-foreground'
        ]"
        @click="selectTheme(option.value)"
      >
        <component :is="option.icon" class="h-4 w-4 shrink-0" />
      </button>
    </div>

    <template v-else>
      <button
        type="button"
        class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-muted/70 hover:text-foreground focus-ring"
        :title="t(activeOption.labelKey)"
        :aria-label="t(activeOption.labelKey)"
        :aria-expanded="open"
        @click.stop="open = !open"
      >
        <component :is="activeOption.icon" class="h-4 w-4 shrink-0" />
      </button>

      <Transition
        enter-active-class="transition duration-150 ease-out"
        enter-from-class="translate-y-1 opacity-0"
        enter-to-class="translate-y-0 opacity-100"
        leave-active-class="transition duration-100 ease-in"
        leave-from-class="translate-y-0 opacity-100"
        leave-to-class="translate-y-1 opacity-0"
      >
        <div v-if="open" class="absolute right-0 top-full z-50 mt-2 w-44 rounded-xl border border-border bg-popover p-1 text-popover-foreground shadow-2xl shadow-slate-950/10 dark:shadow-black/25">
          <button
            v-for="option in options"
            :key="option.value"
            type="button"
            :class="
              cn(
                'flex w-full items-center justify-between gap-3 rounded-lg px-3 py-2 text-left text-sm transition-colors focus-ring',
                colorMode.preference === option.value ? 'bg-primary/10 text-foreground' : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              )
            "
            @click="selectTheme(option.value)"
          >
            <span class="inline-flex min-w-0 items-center gap-2">
              <component :is="option.icon" class="h-4 w-4 shrink-0" />
              <span class="truncate">{{ t(option.labelKey) }}</span>
            </span>
            <Check v-if="colorMode.preference === option.value" class="h-4 w-4 shrink-0 text-primary" />
          </button>
        </div>
      </Transition>
    </template>
  </div>
</template>
