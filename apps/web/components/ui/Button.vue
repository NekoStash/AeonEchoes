<script setup lang="ts">
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '~/lib/utils'

const buttonVariants = cva(
  'inline-flex min-w-0 shrink-0 items-center justify-center gap-2 rounded-xl text-center text-sm font-medium leading-5 transition-all focus-ring disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground shadow-sm hover:bg-primary/90',
        secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/85',
        outline: 'border border-border bg-background text-foreground shadow-sm hover:bg-muted/70',
        ghost: 'text-muted-foreground hover:bg-muted/65 hover:text-foreground',
        destructive: 'bg-destructive text-destructive-foreground shadow-sm hover:bg-destructive/90',
        archive: 'border border-border bg-card text-card-foreground shadow-sm hover:bg-muted/70'
      },
      size: {
        sm: 'min-h-8 px-3 py-1.5 text-xs',
        md: 'min-h-10 px-4 py-2',
        lg: 'min-h-12 px-6 py-3 text-base',
        icon: 'h-10 w-10'
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

const props = withDefaults(
  defineProps<{
    variant?: ButtonVariants['variant']
    size?: ButtonVariants['size']
    type?: 'button' | 'submit' | 'reset'
    to?: string
    disabled?: boolean
    class?: string
  }>(),
  {
    type: 'button',
    variant: 'default',
    size: 'md'
  }
)

function handleDisabledLinkClick(event: MouseEvent) {
  if (!props.disabled) return
  event.preventDefault()
  event.stopPropagation()
}
</script>

<template>
  <NuxtLink
    v-if="to"
    v-bind="$attrs"
    :to="to"
    :aria-disabled="disabled ? 'true' : undefined"
    :tabindex="disabled ? -1 : undefined"
    :class="cn(buttonVariants({ variant, size }), disabled && 'cursor-not-allowed opacity-50', props.class)"
    @click.capture="handleDisabledLinkClick"
  >
    <slot />
  </NuxtLink>
  <button v-else v-bind="$attrs" :type="type" :disabled="disabled" :class="cn(buttonVariants({ variant, size }), props.class)">
    <slot />
  </button>
</template>
