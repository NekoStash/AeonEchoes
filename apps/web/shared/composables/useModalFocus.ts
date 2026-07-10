import { nextTick, onBeforeUnmount, watch, type Ref } from 'vue'

const focusableSelector = [
  '[autofocus]',
  'button:not([disabled])',
  '[href]',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])'
].join(',')

const overlayStack: symbol[] = []
const inertSnapshots = new Map<HTMLElement, { inert: boolean; ariaHidden: string | null }>()

function setPageInert(container: HTMLElement, inert: boolean) {
  const bodyChildren = Array.from(document.body.children).filter((element): element is HTMLElement => element instanceof HTMLElement)
  if (inert) {
    for (const element of bodyChildren) {
      if (element === container || element.contains(container) || element.dataset.aeonOverlayLayer === 'true') continue
      if (!inertSnapshots.has(element)) {
        inertSnapshots.set(element, { inert: element.inert, ariaHidden: element.getAttribute('aria-hidden') })
      }
      element.inert = true
      element.setAttribute('aria-hidden', 'true')
    }
    return
  }

  for (const [element, snapshot] of inertSnapshots) {
    if (!element.isConnected) continue
    element.inert = snapshot.inert
    if (snapshot.ariaHidden === null) element.removeAttribute('aria-hidden')
    else element.setAttribute('aria-hidden', snapshot.ariaHidden)
  }
  inertSnapshots.clear()
}

function focusableElements(container: HTMLElement) {
  return Array.from(container.querySelectorAll<HTMLElement>(focusableSelector)).filter((element) => {
    return !element.hidden && element.getAttribute('aria-hidden') !== 'true'
  })
}

function lockBodyScroll() {
  const body = document.body
  const lockCount = Number(body.dataset.aeonScrollLocks || '0')
  if (lockCount === 0) {
    body.dataset.aeonOriginalOverflow = body.style.overflow
    body.style.overflow = 'hidden'
  }
  body.dataset.aeonScrollLocks = String(lockCount + 1)
}

function unlockBodyScroll() {
  const body = document.body
  const lockCount = Number(body.dataset.aeonScrollLocks || '0')
  const nextCount = Math.max(0, lockCount - 1)
  if (nextCount > 0) {
    body.dataset.aeonScrollLocks = String(nextCount)
    return
  }

  body.style.overflow = body.dataset.aeonOriginalOverflow || ''
  delete body.dataset.aeonOriginalOverflow
  delete body.dataset.aeonScrollLocks
}

export function useModalFocus(open: Ref<boolean>, container: Ref<HTMLElement | null>, requestClose: () => void) {
  const overlayId = Symbol('aeon-overlay')
  let restoreFocusElement: HTMLElement | null = null
  let active = false

  function isTopOverlay() {
    return overlayStack.at(-1) === overlayId
  }

  function focusInitialElement() {
    const root = container.value
    if (!root) {
      console.error('[AeonEchoes UI] Overlay opened without a mounted dialog container.')
      return
    }
    const [first] = focusableElements(root)
    ;(first || root).focus({ preventScroll: true })
  }

  function handleKeydown(event: KeyboardEvent) {
    if (!open.value || !isTopOverlay()) return

    if (event.key === 'Escape') {
      event.preventDefault()
      requestClose()
      return
    }

    if (event.key !== 'Tab') return
    const root = container.value
    if (!root) return
    const elements = focusableElements(root)
    if (elements.length === 0) {
      event.preventDefault()
      root.focus({ preventScroll: true })
      return
    }

    const first = elements[0]
    const last = elements[elements.length - 1]
    const current = document.activeElement
    if (event.shiftKey && (current === first || !root.contains(current))) {
      event.preventDefault()
      last?.focus()
    } else if (!event.shiftKey && (current === last || !root.contains(current))) {
      event.preventDefault()
      first?.focus()
    }
  }

  function activate() {
    if (active || typeof document === 'undefined') return
    active = true
    restoreFocusElement = document.activeElement instanceof HTMLElement ? document.activeElement : null
    overlayStack.push(overlayId)
    lockBodyScroll()
    document.addEventListener('keydown', handleKeydown)
  }

  function deactivate(restoreFocus = true) {
    if (!active || typeof document === 'undefined') return
    active = false
    const stackIndex = overlayStack.lastIndexOf(overlayId)
    if (stackIndex >= 0) overlayStack.splice(stackIndex, 1)
    document.removeEventListener('keydown', handleKeydown)
    unlockBodyScroll()
    if (overlayStack.length === 0) setPageInert(container.value || document.body, false)
    if (restoreFocus && restoreFocusElement?.isConnected) {
      restoreFocusElement.focus({ preventScroll: true })
    }
    restoreFocusElement = null
  }

  watch(
    open,
    async (isOpen) => {
      if (typeof document === 'undefined') return
      if (isOpen) {
        activate()
        await nextTick()
        const root = container.value
        if (root) setPageInert(root, true)
        focusInitialElement()
      } else {
        await nextTick()
        deactivate()
      }
    },
    { immediate: true, flush: 'post' }
  )

  onBeforeUnmount(() => deactivate(false))

  return { focusInitialElement }
}
