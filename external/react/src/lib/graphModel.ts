// Pure graph-view logic: colors, filtering, focus highlighting. Kept
// framework-free so it stays unit-testable without sigma/canvas.

export type GraphKind = 'note' | 'person'
export type ConnectionType = 'similar' | 'person' | 'amount' | 'contradiction' | 'follow_up' | 'revisit'

export interface GraphNode {
  id: string
  kind: GraphKind
  name: string
}

export interface GraphEdge {
  source: string
  target: string
  types?: string[]
}

export interface GraphData {
  nodes: GraphNode[]
  edges: GraphEdge[]
}

// hex -> rgba with alpha: edges render whisper-thin and dim at rest,
// brightening only on focus — the signature of polished graph views
export function withAlpha(hex: string, alpha: number): string {
  const h = hex.replace('#', '')
  if (h.length !== 6) return hex
  const r = parseInt(h.slice(0, 2), 16)
  const g = parseInt(h.slice(2, 4), 16)
  const b = parseInt(h.slice(4, 6), 16)
  return `rgba(${r},${g},${b},${alpha})`
}

// The product's type color language (matches linked-note badges)
export const EDGE_COLORS: Record<string, string> = {
  similar: '#3ddc84',
  person: '#c9933a',
  amount: '#e8b86d',
  contradiction: '#ff8a5c',
  follow_up: '#ffd166',
  revisit: '#8ab4ff',
}

export const NODE_COLORS: Record<GraphKind, string> = {
  person: '#c9933a',
  note: '#8a8f98',
}

export const CONNECTION_TYPE_LABELS: Record<string, string> = {
  similar: 'similar',
  person: 'person',
  amount: 'amount',
  contradiction: 'contradiction',
  follow_up: 'follow-up',
  revisit: 'revisited',
}

// Primary edge color = the first type's color; person-mention edges inherit
// the person gold. Un-typed edges get a dim neutral.
export function edgeColor(edge: GraphEdge): string {
  if (!edge.types || edge.types.length === 0) return 'rgba(245,245,245,0.25)'
  return EDGE_COLORS[edge.types[0]] || 'rgba(245,245,245,0.3)'
}

// True if any of the edge's types survive the active filter set.
export function edgeMatchesFilter(edge: GraphEdge, active: Set<string>): boolean {
  if (active.size === 0) return true // no filter = everything
  if (!edge.types || edge.types.length === 0) return active.has('person')
  return edge.types.some((t) => active.has(t as ConnectionType))
}

// Node ids adjacent to the given node through visible edges.
export function neighborIds(nodeId: string, edges: GraphEdge[]): Set<string> {
  const out = new Set<string>()
  for (const e of edges) {
    if (e.source === nodeId) out.add(e.target)
    else if (e.target === nodeId) out.add(e.source)
  }
  return out
}

// Filtered view: drop edges that don't match, drop nodes left with no
// edges (keeps the canvas readable when filters are narrow).
export function filterGraph(
  data: GraphData,
  active: Set<string>,
): GraphData {
  if (active.size === 0) return data
  const edges = data.edges.filter((e) => edgeMatchesFilter(e, active))
  const keep = new Set<string>()
  for (const e of edges) {
    keep.add(e.source)
    keep.add(e.target)
  }
  return {
    nodes: data.nodes.filter((n) => keep.has(n.id)),
    edges,
  }
}

// Person nodes carry their label always; notes only when focused —
// 500 unlabeled notes would render the canvas unreadable.
export function nodeLabel(node: GraphNode, showNoteLabels: boolean): string {
  if (node.kind === 'person') return node.name
  return showNoteLabels ? node.name : ''
}

export function personCount(data: GraphData): number {
  return data.nodes.filter((n) => n.kind === 'person').length
}
