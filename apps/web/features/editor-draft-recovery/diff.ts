export type DiffLineKind = 'same' | 'added' | 'removed'

export interface DiffLine {
  kind: DiffLineKind
  text: string
}

export function buildLineDiff(backendContent: string, localContent: string): DiffLine[] {
  const left = backendContent.split('\n')
  const right = localContent.split('\n')
  const matrix = Array.from({ length: left.length + 1 }, () => Array<number>(right.length + 1).fill(0))

  for (let leftIndex = left.length - 1; leftIndex >= 0; leftIndex -= 1) {
    for (let rightIndex = right.length - 1; rightIndex >= 0; rightIndex -= 1) {
      matrix[leftIndex]![rightIndex] = left[leftIndex] === right[rightIndex]
        ? matrix[leftIndex + 1]![rightIndex + 1]! + 1
        : Math.max(matrix[leftIndex + 1]![rightIndex]!, matrix[leftIndex]![rightIndex + 1]!)
    }
  }

  const result: DiffLine[] = []
  let leftIndex = 0
  let rightIndex = 0
  while (leftIndex < left.length && rightIndex < right.length) {
    if (left[leftIndex] === right[rightIndex]) {
      result.push({ kind: 'same', text: left[leftIndex]! })
      leftIndex += 1
      rightIndex += 1
    } else if (matrix[leftIndex + 1]![rightIndex]! >= matrix[leftIndex]![rightIndex + 1]!) {
      result.push({ kind: 'removed', text: left[leftIndex]! })
      leftIndex += 1
    } else {
      result.push({ kind: 'added', text: right[rightIndex]! })
      rightIndex += 1
    }
  }
  while (leftIndex < left.length) result.push({ kind: 'removed', text: left[leftIndex++]! })
  while (rightIndex < right.length) result.push({ kind: 'added', text: right[rightIndex++]! })
  return result
}
