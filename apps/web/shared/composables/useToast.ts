import { readonly, ref } from 'vue'

export type ToastTone = 'info' | 'success' | 'warning' | 'danger'

export interface ToastMessage {
  id: number
  title: string
  description?: string
  tone: ToastTone
  duration: number
}

export interface ToastInput {
  title: string
  description?: string
  tone?: ToastTone
  duration?: number
  cause?: unknown
}

const messages = ref<ToastMessage[]>([])
const timers = new Map<number, ReturnType<typeof setTimeout>>()
let nextId = 1

function remove(id: number) {
  const timer = timers.get(id)
  if (timer) clearTimeout(timer)
  timers.delete(id)
  messages.value = messages.value.filter((message) => message.id !== id)
}

function push(input: ToastInput) {
  const title = input.title.trim()
  if (!title) {
    const error = new Error('Toast title must not be empty.')
    console.error('[AeonEchoes UI] Invalid toast payload.', error, input)
    throw error
  }

  if (input.tone === 'danger') {
    console.error(`[AeonEchoes UI] ${title}`, input.cause || input.description || title)
  }

  const message: ToastMessage = {
    id: nextId++,
    title,
    description: input.description?.trim() || undefined,
    tone: input.tone || 'info',
    duration: Math.max(0, input.duration ?? 5000)
  }
  messages.value = [...messages.value, message]

  if (message.duration > 0 && import.meta.client) {
    timers.set(message.id, setTimeout(() => remove(message.id), message.duration))
  }

  return message.id
}

export function useToast() {
  return {
    messages: readonly(messages),
    push,
    remove,
    info: (title: string, description?: string) => push({ title, description, tone: 'info' }),
    success: (title: string, description?: string) => push({ title, description, tone: 'success' }),
    warning: (title: string, description?: string) => push({ title, description, tone: 'warning' }),
    error: (title: string, description?: string, cause?: unknown) => push({ title, description, cause, tone: 'danger', duration: 0 })
  }
}
