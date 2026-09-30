import { Node } from '@/types'

export type SortBy = 'name' | 'cpu' | 'memory' | 'disk' | 'status'
export type SortOrder = 'asc' | 'desc'
export type StatusFilter = 'all' | 'active' | 'inactive' | 'error'

export interface FilterOptions {
  status: StatusFilter
  sortBy: SortBy
  sortOrder: SortOrder
  search: string
}

export const filterNodes = (nodes: Node[], options: FilterOptions): Node[] => {
  let filtered = [...nodes]

  // Filter by status
  if (options.status !== 'all') {
    filtered = filtered.filter((node) => node.status === options.status)
  }

  // Filter by search term
  if (options.search) {
    const term = options.search.toLowerCase()
    filtered = filtered.filter(
      (node) =>
        node.name.toLowerCase().includes(term) ||
        node.id.toLowerCase().includes(term)
    )
  }

  // Sort
  filtered.sort((a, b) => {
    let aVal: number | string
    let bVal: number | string

    switch (options.sortBy) {
      case 'name':
        aVal = a.name
        bVal = b.name
        break
      case 'cpu':
        aVal = a.cpu.percent
        bVal = b.cpu.percent
        break
      case 'memory':
        aVal = a.memory.percent
        bVal = b.memory.percent
        break
      case 'disk':
        aVal = a.disk.percent
        bVal = b.disk.percent
        break
      case 'status':
        aVal = a.status
        bVal = b.status
        break
    }

    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return options.sortOrder === 'asc'
        ? aVal.localeCompare(bVal)
        : bVal.localeCompare(aVal)
    }

    return options.sortOrder === 'asc'
      ? (aVal as number) - (bVal as number)
      : (bVal as number) - (aVal as number)
  })

  return filtered
}
