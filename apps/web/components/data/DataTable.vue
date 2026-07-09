<script setup lang="ts">
import { cn } from '~/lib/utils'

export interface DataTableColumn {
  key: string
  label: string
  class?: string
  headerClass?: string
  align?: 'left' | 'center' | 'right'
}

type Row = Record<string, unknown>

const props = withDefaults(
  defineProps<{
    columns: DataTableColumn[]
    rows: Row[]
    rowKey?: string
    density?: 'compact' | 'comfortable' | 'relaxed'
    caption?: string
    class?: string
  }>(),
  {
    rowKey: 'id',
    density: 'comfortable',
    caption: undefined
  }
)

const emit = defineEmits<{
  rowClick: [row: Row]
}>()

const densityClass = computed(() => {
  const densities = {
    compact: 'px-3 py-2 text-xs',
    comfortable: 'px-4 py-3 text-sm',
    relaxed: 'px-5 py-4 text-sm'
  }
  return densities[props.density]
})

function cellValue(row: Row, key: string) {
  return key.split('.').reduce<unknown>((value, segment) => {
    if (value && typeof value === 'object' && segment in value) {
      return (value as Row)[segment]
    }
    return undefined
  }, row)
}

function stableRowKey(row: Row, index: number) {
  const value = cellValue(row, props.rowKey)
  return typeof value === 'string' || typeof value === 'number' ? value : index
}

function alignClass(align?: DataTableColumn['align']) {
  if (align === 'center') return 'text-center'
  if (align === 'right') return 'text-right'
  return 'text-left'
}
</script>

<template>
  <div :class="cn('min-w-0 overflow-hidden rounded-2xl border border-border bg-surface shadow-sm', props.class)">
    <div class="overflow-x-auto subtle-scrollbar">
      <table class="min-w-full border-separate border-spacing-0 text-left">
        <caption v-if="caption" class="sr-only">{{ caption }}</caption>
        <thead class="bg-surface-muted text-xs uppercase tracking-[0.14em] text-muted-foreground">
          <tr>
            <th
              v-for="column in columns"
              :key="column.key"
              scope="col"
              :class="cn('border-b border-border font-medium', densityClass, alignClass(column.align), column.headerClass)"
            >
              {{ column.label }}
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr
            v-for="(row, index) in rows"
            :key="stableRowKey(row, index)"
            class="transition-colors hover:bg-surface-muted/70"
            @click="emit('rowClick', row)"
          >
            <td
              v-for="column in columns"
              :key="column.key"
              :class="cn('align-top text-foreground', densityClass, alignClass(column.align), column.class)"
            >
              <slot name="cell" :row="row" :column="column" :value="cellValue(row, column.key)">
                {{ cellValue(row, column.key) ?? '—' }}
              </slot>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <DataEmptyState v-if="rows.length === 0" class="m-4" />
  </div>
</template>
