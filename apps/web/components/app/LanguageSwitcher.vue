<script setup lang="ts">
import { Check, Languages } from '@lucide/vue'
import { cn } from '~/lib/utils'

type LocaleCode = 'zh-CN' | 'en-US'

const props = withDefaults(
  defineProps<{
    compact?: boolean
  }>(),
  {
    compact: false
  }
)

const { locale, locales, setLocale, t } = useI18n()

const open = ref(false)
const root = ref<HTMLElement | null>(null)

const availableLocales = computed(() => locales.value.map((item) => (typeof item === 'string' ? { code: item, name: item } : item)))
const localeOptions = computed(() => availableLocales.value.map((item) => ({ label: item.name || item.code, value: item.code })))
const selectedLocale = computed({
  get: () => locale.value,
  set: (value: string) => selectLocale(value)
})
const activeLocaleLabel = computed(() => localeOptions.value.find((item) => item.value === locale.value)?.label || locale.value)

function isLocaleCode(value: string): value is LocaleCode {
  return value === 'zh-CN' || value === 'en-US'
}

function selectLocale(value: string) {
  if (!isLocaleCode(value)) {
    console.error('Unsupported locale selected', value)
    return
  }
  setLocale(value)
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
  <div ref="root" :class="cn('relative', compact ? 'inline-flex' : 'w-36')">
    <span class="sr-only">{{ t('language.label') }}</span>
    <UiSelect v-if="!compact" v-model="selectedLocale" :options="localeOptions" :aria-label="t('language.label')" />

    <template v-else>
      <button
        type="button"
        class="focus-ring flex h-10 w-10 shrink-0 items-center justify-center border border-border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        :title="`${t('language.label')}: ${activeLocaleLabel}`"
        :aria-label="t('language.label')"
        :aria-expanded="open"
        @click.stop="open = !open"
      >
        <Languages class="h-4 w-4 shrink-0" />
      </button>

      <Transition
        enter-active-class="transition duration-150 ease-out"
        enter-from-class="translate-y-1 opacity-0"
        enter-to-class="translate-y-0 opacity-100"
        leave-active-class="transition duration-100 ease-in"
        leave-from-class="translate-y-0 opacity-100"
        leave-to-class="translate-y-1 opacity-0"
      >
        <div v-if="open" class="absolute right-0 top-full z-50 mt-1 w-44 border border-border bg-popover p-1 text-popover-foreground" role="menu">
          <button
            v-for="option in localeOptions"
            :key="option.value"
            type="button"
            role="menuitemradio"
            :aria-checked="option.value === locale"
            :class="
              cn(
                'focus-ring flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm transition-colors',
                option.value === locale ? 'bg-primary/10 text-foreground' : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              )
            "
            @click="selectLocale(option.value)"
          >
            <span class="truncate">{{ option.label }}</span>
            <Check v-if="option.value === locale" class="h-4 w-4 shrink-0 text-primary" />
          </button>
        </div>
      </Transition>
    </template>
  </div>
</template>
