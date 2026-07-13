<script setup lang="ts">
import { Maximize2, ZoomIn, ZoomOut } from '@lucide/vue'
import type { Core, ElementDefinition } from 'cytoscape'
import type { GraphEdge, GraphNode } from '~/lib/types'
import { cssColor as formatCssColor } from './graph-view'

const props = defineProps<{
  nodes: GraphNode[]
  edges: GraphEdge[]
  selectedId?: string
}>()

const emit = defineEmits<{
  selectNode: [id: string]
  selectEdge: [id: string]
  error: [message: string]
}>()

const { t } = useI18n()
const container = ref<HTMLElement | null>(null)
const initializing = ref(true)
let cy: Core | null = null

const elements = computed<ElementDefinition[]>(() => [
  ...props.nodes.map((node) => ({
    data: {
      id: node.id,
      label: node.label,
      type: node.type,
      status: node.status,
      importance: node.importance
    }
  })),
  ...props.edges.map((edge) => ({
    data: {
      id: edge.id,
      source: edge.source,
      target: edge.target,
      label: edge.label,
      type: edge.type,
      weight: edge.weight
    }
  }))
])

onMounted(async () => {
  await render()
})

watch(elements, async () => {
  await render()
})

watch(() => props.selectedId, syncSelection)

onBeforeUnmount(() => {
  cy?.destroy()
  cy = null
})

async function render() {
  if (!import.meta.client || !container.value) return
  initializing.value = true
  try {
    const cytoscape = (await import('cytoscape')).default
    if (!cy) {
      cy = cytoscape({
        container: container.value,
        elements: elements.value,
        minZoom: 0.3,
        maxZoom: 2.5,
        wheelSensitivity: 0.24,
        style: graphStyle(),
        layout: createLayout()
      })
      cy.on('tap', 'node', (event) => emit('selectNode', event.target.id()))
      cy.on('tap', 'edge', (event) => emit('selectEdge', event.target.id()))
    } else {
      cy.elements().remove()
      cy.add(elements.value)
      cy.style(graphStyle()).update()
      cy.layout(createLayout()).run()
    }
    syncSelection()
  } catch (error) {
    console.error('[graph-explore] Cytoscape failed to initialize.', error)
    emit('error', error instanceof Error ? error.message : t('graph.errors.cytoscapeFailed'))
  } finally {
    initializing.value = false
  }
}

function createLayout() {
  return {
    name: 'cose',
    animate: false,
    fit: true,
    padding: 48,
    nodeRepulsion: 7000,
    idealEdgeLength: 128
  } as const
}

function syncSelection() {
  if (!cy) return
  cy.elements().removeClass('is-selected')
  if (props.selectedId) cy.$id(props.selectedId).addClass('is-selected')
}

function zoomBy(factor: number) {
  if (!cy) return
  cy.zoom({ level: Math.min(2.5, Math.max(0.3, cy.zoom() * factor)), renderedPosition: { x: cy.width() / 2, y: cy.height() / 2 } })
}

function fitGraph() {
  cy?.fit(undefined, 36)
}

function graphStyle() {
  const foreground = cssColor('--foreground', 'hsl(220, 15%, 12%)')
  const background = cssColor('--background', 'hsl(42, 20%, 96%)')
  const muted = cssColor('--muted-foreground', 'hsl(218, 9%, 40%)')
  const border = cssColor('--border', 'hsl(38, 10%, 75%)')
  const danger = cssColor('--state-danger', 'hsl(3, 57%, 43%)')
  const info = cssColor('--state-info', 'hsl(212, 72%, 42%)')
  const success = cssColor('--state-success', 'hsl(148, 42%, 32%)')
  const warning = cssColor('--state-warning', 'hsl(34, 78%, 42%)')
  return [
    {
      selector: 'node',
      style: {
        label: 'data(label)',
        color: foreground,
        'font-size': 11,
        'font-weight': 600,
        'text-outline-color': background,
        'text-outline-width': 3,
        'background-color': muted,
        'border-color': border,
        'border-width': 2,
        // 后端 importance 常见为 0..100 整数（如 50），不是 0..1 小数
        width: 'mapData(importance, 0, 100, 32, 58)',
        height: 'mapData(importance, 0, 100, 32, 58)'
      }
    },
    { selector: 'node[type = "character"]', style: { 'background-color': info } },
    { selector: 'node[type = "event"]', style: { 'background-color': danger } },
    { selector: 'node[type = "rule"]', style: { 'background-color': success } },
    { selector: 'node[type = "chapter"]', style: { 'background-color': foreground } },
    { selector: 'node[type = "location"]', style: { 'background-color': warning } },
    { selector: 'node[type = "clue"]', style: { 'background-color': cssColor('--state-info-foreground', 'hsl(212, 80%, 28%)') } },
    { selector: 'node[status = "conflict"]', style: { 'border-color': danger, 'border-width': 4 } },
    { selector: 'node.is-selected', style: { 'border-color': foreground, 'border-width': 6 } },
    {
      selector: 'edge',
      style: {
        label: 'data(label)',
        color: muted,
        'font-size': 9,
        'text-background-color': background,
        'text-background-opacity': 0.88,
        width: 'mapData(weight, 0, 1, 1, 4)',
        'line-color': border,
        'target-arrow-color': muted,
        'target-arrow-shape': 'triangle',
        'curve-style': 'bezier'
      }
    },
    { selector: 'edge[type = "contradicts"]', style: { 'line-color': danger, 'target-arrow-color': danger, 'line-style': 'dashed' } },
    { selector: 'edge.is-selected', style: { 'line-color': foreground, 'target-arrow-color': foreground, width: 5 } }
  ]
}

function cssColor(token: string, fallback: string) {
  if (!import.meta.client) return fallback
  return formatCssColor(token, fallback, (name) => getComputedStyle(document.documentElement).getPropertyValue(name))
}
</script>

<template>
  <div class="relative min-h-[34rem] overflow-hidden border border-border bg-surface-sunken lg:min-h-[44rem]">
    <div ref="container" class="absolute inset-0" />
    <div class="absolute right-3 top-3 z-10 flex border border-border bg-background">
      <button type="button" class="focus-ring flex h-10 w-10 items-center justify-center border-r border-border hover:bg-muted" :aria-label="t('graph.canvas.zoomIn')" @click="zoomBy(1.2)"><ZoomIn class="h-4 w-4" /></button>
      <button type="button" class="focus-ring flex h-10 w-10 items-center justify-center border-r border-border hover:bg-muted" :aria-label="t('graph.canvas.zoomOut')" @click="zoomBy(0.82)"><ZoomOut class="h-4 w-4" /></button>
      <button type="button" class="focus-ring flex h-10 w-10 items-center justify-center hover:bg-muted" :aria-label="t('graph.canvas.fit')" @click="fitGraph"><Maximize2 class="h-4 w-4" /></button>
    </div>
    <div v-if="initializing" class="absolute inset-0 flex items-center justify-center bg-background/80 text-sm font-semibold text-muted-foreground">
      {{ t('graph.states.canvasLoadingTitle') }}
    </div>
  </div>
</template>
