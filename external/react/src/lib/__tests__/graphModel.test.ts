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

import { withAlpha } from '../graphModel'

describe('withAlpha', () => {
  it('converts hex to rgba with alpha', () => {
    expect(withAlpha('#ff8a5c', 0.35)).toBe('rgba(255,138,92,0.35)')
  })
  it('passes through non-hex values untouched', () => {
    expect(withAlpha('rgba(1,2,3,0.5)', 0.9)).toBe('rgba(1,2,3,0.5)')
  })
})

import { nodeSize, isHub, NOTE_TYPE_COLORS } from '../graphModel'

describe('node sizing', () => {
  it('persons are a fixed larger size', () => {
    expect(nodeSize({ id: 'p', kind: 'person', name: 'A' }, 20)).toBe(8)
  })
  it('notes grow with degree, capped', () => {
    const n = { id: 'x', kind: 'note' as const, name: 'x' }
    expect(nodeSize(n, 0)).toBe(4)
    expect(nodeSize(n, 5)).toBe(6.5)
    expect(nodeSize(n, 100)).toBe(10)
  })
  it('hub = person or degree >= 5', () => {
    expect(isHub({ id: 'p', kind: 'person', name: 'A' }, 0)).toBe(true)
    expect(isHub({ id: 'n', kind: 'note', name: 'N' }, 4)).toBe(false)
    expect(isHub({ id: 'n', kind: 'note', name: 'N' }, 5)).toBe(true)
  })
  it('note types have distinct colors', () => {
    const set = new Set(Object.values(NOTE_TYPE_COLORS))
    expect(set.size).toBe(4)
  })
})

import { revealProgress } from '../graphModel'

describe('revealProgress', () => {
  it('advances by total/90 per frame for big graphs', () => {
    expect(revealProgress(0, 90)).toBeCloseTo(1, 5)
  })
  it('never advances slower than half a node per frame', () => {
    expect(revealProgress(0, 10)).toBe(0.5)
  })
  it('caps at total + ease tail', () => {
    expect(revealProgress(200, 90)).toBe(98)
  })
  it('handles empty graphs', () => {
    expect(revealProgress(5, 0)).toBe(0)
  })
})
