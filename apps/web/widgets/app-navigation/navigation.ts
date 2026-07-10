import type { Component } from 'vue'

export interface AppNavigationItem {
  label: string
  to: string
  icon: Component
  exact?: boolean
}

export interface AppNavigationGroup {
  label: string
  items: AppNavigationItem[]
}

export function isRouteActive(currentPath: string, targetPath: string, exact = false) {
  if (exact || targetPath === '/') return currentPath === targetPath
  return currentPath === targetPath || currentPath.startsWith(`${targetPath}/`)
}
