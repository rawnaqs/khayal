import { describe, it, expect } from 'vitest'
import {
  edgeColor,
  edgeMatchesFilter,
  neighborIds,
  filterGraph,
  personCount,
  EDGE_COLORS,
} from '../graphModel'
import type { GraphEdge } from '../graphModel'

const edge = (types?: string[]): GraphEdge => ({
  source: 'a',
  target: 'b',
  types,
})

describe('edge colors', () => {
  it('contradiction edges get the warning orange', () => {
    expect(edgeColor(edge(['contradiction']))).toBe('#ff8a5c')
  })
  it('un-typed edges are dim neutral', () => {
    expect(edgeColor(edge(undefined))).toBe('rgba(245,245,245,0.25)')
  })
  it('first type wins the color', () => {
    expect(edgeColor(edge(['revisit', 'similar']))).toBe('#8ab4ff')
  })
})

describe('filters', () => {
  it('empty filter = everything visible', () => {
    expect(edgeMatchesFilter(edge(['contradiction']), new Set())).toBe(true)
  })
  it('edge survives when any type matches', () => {
    expect(edgeMatchesFilter(edge(['similar', 'person']), new Set(['person' as const]))).toBe(true)
  })
  it('un-typed edges belong to the person filter', () => {
    expect(edgeMatchesFilter(edge(undefined), new Set(['person' as const]))).toBe(true)
    expect(edgeMatchesFilter(edge(undefined), new Set(['similar' as const]))).toBe(false)
  })

  it('filterGraph drops unmatching edges and orphaned nodes', () => {
    const data = {
      nodes: [
        { id: 'a', kind: 'note' as const, name: 'a' },
        { id: 'b', kind: 'note' as const, name: 'b' },
        { id: 'c', kind: 'note' as const, name: 'c' },
      ],
      edges: [
        { source: 'a', target: 'b', types: ['similar'] },
        { source: 'b', target: 'c', types: ['contradiction'] },
      ],
    }
    const filtered = filterGraph(data, new Set(['contradiction' as const]))
    expect(filtered.edges.length).toBe(1)
    expect(filtered.nodes.map((n) => n.id).sort()).toEqual(['b', 'c'])
  })
})

describe('neighbors', () => {
  it('collects both directions', () => {
    const n = neighborIds('a', [
      { source: 'a', target: 'b', types: [] },
      { source: 'c', target: 'a', types: [] },
      { source: 'x', target: 'y', types: [] },
    ])
    expect(n.has('b')).toBe(true)
    expect(n.has('c')).toBe(true)
    expect(n.has('x')).toBe(false)
  })
})

describe('personCount', () => {
  it('counts only people', () => {
    expect(personCount({
      nodes: [
        { id: 'p1', kind: 'person', name: 'A' },
        { id: 'n1', kind: 'note', name: 'N' },
      ],
      edges: [],
    })).toBe(1)
  })
})
