import { createApiClient } from '~/lib/api'

function resolveLocale() {
  try {
    const { locale } = useI18n()
    return locale.value
  } catch {
    const nuxtApp = useNuxtApp()
    const i18n = nuxtApp.$i18n as { locale?: string | { value?: string } } | undefined
    const appLocale = i18n?.locale
    if (typeof appLocale === 'string') return appLocale
    if (appLocale && typeof appLocale === 'object' && typeof appLocale.value === 'string') {
      return appLocale.value
    }
    return 'zh-CN'
  }
}

export function useApi() {
  const config = useRuntimeConfig()
  return createApiClient(config.public.apiBase, resolveLocale())
}
