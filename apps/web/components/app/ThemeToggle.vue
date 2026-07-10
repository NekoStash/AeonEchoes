<script setup lang="ts">
import { Check, Monitor, Moon, Sun } from '@lucide/vue'
import { cn } from '~/lib/utils'

const props = withDefaults(defineProps<{
  compact?: boolean
}>(), {
  compact: false
})

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
  if (!root.value?.contains(event.target as Node)) open.value = false
}

onMounted(() => document.addEventListener('click', handleDocumentClick))
onBeforeUnmount(() => document.removeEventListener('click', handleDocumentClick))
</script>

<template>
  <div ref="root" class="relative inline-flex">
    <div v-if="!compact" class="inline-flex border border-border bg-card p-1" role="group" :aria-label="t('theme.label')">
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        :title="t(option.labelKey)"
        :aria-label="t(option.labelKey)"
        :aria-pressed="colorMode.preference === option.value"
        :class="cn('focus-ring inline-flex h-8 w-8 items-center justify-center text-muted-foreground transition-colors hover:text-foreground', colorMode.preference === option.value && 'bg-foreground text-background')"
        @click="selectTheme(option.value)"
      >
        <component :is="option.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
      </button>
    </div>

    <template v-else>
      <button
        type="button"
        class="focus-ring flex h-10 w-10 shrink-0 items-center justify-center border border-border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        :title="t(activeOption.labelKey)"
        :aria-label="`${t('theme.label')}: ${t(activeOption.labelKey)}`"
        :aria-expanded="open"
        aria-haspopup="menu"
        @click.stop="open = !open"
      >
        <component :is="activeOption.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
      </button>

      <div v-if="open" class="absolute right-0 top-full z-50 mt-1 w-44 border border-border bg-popover p-1 text-popover-foreground" role="menu">
        <button
          v-for="option in options"
          :key="option.value"
          type="button"
          role="menuitemradio"
          :aria-checked="colorMode.preference === option.value"
          :class="cn('focus-ring flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm transition-colors', colorMode.preference === option.value ? 'bg-foreground text-background' : 'text-muted-foreground hover:bg-muted hover:text-foreground')"
          @click="selectTheme(option.value)"
        >
          <span class="inline-flex min-w-0 items-center gap-2">
            <component :is="option.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
            <span class="truncate">{{ t(option.labelKey) }}</span>
          </span>
          <Check v-if="colorMode.preference === option.value" class="h-4 w-4 shrink-0" aria-hidden="true" />
        </button>
      </div>
    </template>
  </div>
</template>
