import type { Chapter } from './types'

export function applyCreatedChapter(items: Chapter[], chapter: Chapter): Chapter[] {
  return [...items.filter((item) => item.id !== chapter.id), chapter].sort((left, right) => left.number - right.number)
}

export function applyUpdatedChapter(items: Chapter[], chapter: Chapter): Chapter[] {
  return [...items.filter((item) => item.id !== chapter.id), chapter].sort((left, right) => left.number - right.number)
}
