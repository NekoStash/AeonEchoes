<script setup lang="ts">
import { LoaderCircle } from '@lucide/vue'
import { cva, type VariantProps } from 'class-variance-authority'
import { computed, useAttrs, watchEffect } from 'vue'
import { cn } from '~/lib/utils'

const buttonVariants = cva(
  'focus-ring inline-flex min-w-0 shrink-0 items-center justify-center gap-2 rounded-md border border-transparent text-center text-sm font-semibold leading-5 transition-colors disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'border-foreground bg-foreground text-background hover:bg-foreground/85',
        secondary: 'border-secondary bg-secondary text-secondary-foreground hover:border-foreground/30 hover:bg-accent',
        outline: 'border-border bg-background text-foreground hover:border-foreground/45 hover:bg-muted',
        ghost: 'text-muted-foreground hover:bg-muted hover:text-foreground',
        destructive: 'border-destructive bg-destructive text-destructive-foreground hover:bg-destructive/85'
      },
      size: {
        sm: 'min-h-8 px-3 py-1.5 text-xs',
        md: 'min-h-10 px-4 py-2',
        lg: 'min-h-12 px-5 py-3 text-base',
        icon: 'h-10 w-10 px-0'
      }
    },
    defaultVariants: {
      variant: 'default',
      size: 'md'
    }
  }
)

type ButtonVariants = VariantProps<typeof buttonVariants>

defineOptions({ inheritAttrs: false })
const attrs = useAttrs()

const props = withDefaults(defineProps<{
  variant?: ButtonVariants['variant']
  size?: ButtonVariants['size']
  type?: 'button' | 'submit' | 'reset'
  to?: string
  disabled?: boolean
  loading?: boolean
  loadingLabel?: string
  iconLabel?: string
  class?: string
}>(), {
  type: 'button',
  variant: 'default',
  size: 'md',
  loading: false,
  loadingLabel: undefined,
  iconLabel: undefined
})

const resolvedAriaLabel = computed(() => props.iconLabel || (attrs['aria-label'] as string | undefined))

watchEffect(() => {
  if (props.size !== 'icon' || resolvedAriaLabel.value?.trim()) return
  const error = new Error('Icon buttons require iconLabel or aria-label.')
  console.error('[AeonEchoes UI] Unnamed icon button.', error)
  throw error
})

function handleDisabledLinkClick(event: MouseEvent) {
  if (!props.disabled && !props.loading) return
  event.preventDefault()
  event.stopPropagation()
}
</script>

<template>
  <NuxtLink
    v-if="to"
    v-bind="$attrs"
    :to="to"
    :aria-disabled="disabled || loading ? 'true' : undefined"
    :aria-busy="loading ? 'true' : undefined"
    :aria-label="resolvedAriaLabel"
    :tabindex="disabled || loading ? -1 : undefined"
    :class="cn(buttonVariants({ variant, size }), (disabled || loading) && 'cursor-not-allowed opacity-50', props.class)"
    @click.capture="handleDisabledLinkClick"
  >
    <LoaderCircle v-if="loading" class="h-4 w-4 shrink-0 animate-spin" aria-hidden="true" />
    <span v-if="loading && loadingLabel" class="sr-only">{{ loadingLabel }}</span>
    <slot />
  </NuxtLink>
  <button
    v-else
    v-bind="$attrs"
    :type="type"
    :disabled="disabled || loading"
    :aria-busy="loading ? 'true' : undefined"
    :aria-label="resolvedAriaLabel"
    :class="cn(buttonVariants({ variant, size }), props.class)"
  >
    <LoaderCircle v-if="loading" class="h-4 w-4 shrink-0 animate-spin" aria-hidden="true" />
    <span v-if="loading && loadingLabel" class="sr-only">{{ loadingLabel }}</span>
    <slot />
  </button>
</template>
