import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import Graph from 'graphology'
import forceAtlas2 from 'graphology-layout-forceatlas2'
import { SigmaContainer, useLoadGraph, useRegisterEvents, useSetSettings } from '@react-sigma/core'
import '@react-sigma/core/lib/style.css'
import { createClient, type GraphNode } from '@/lib/api'
import {
  CONNECTION_TYPE_LABELS,
  EDGE_COLORS,
  NODE_COLORS,
  edgeMatchesFilter,
  neighborIds,
} from '@/lib/graphModel'
import { useVaultLock } from '@/hooks/useVaultLock'
import { cn } from '@/lib/utils'

type RawGraph = { nodes: GraphNode[]; edges: { source: string; target: string; types?: string[] }[] }

function useGraphData() {
  const { token } = useVaultLock()
  const [data, setData] = useState<RawGraph | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setData(await createClient(token).graph())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load graph')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => {
    load()
  }, [load])

  return { data, loading, error, load }
}

// GraphInner: builds the graphology model once per data load, then
// applies reducers for filter/selection state. Loaders must live inside
// SigmaContainer (they need the sigma instance from context).
function GraphInner({
  data,
  edgeFilter,
  selected,
  onNodeClick,
}: {
  data: RawGraph
  edgeFilter: Set<string>
  selected: GraphNode | null
  onNodeClick: (id: string) => void
}) {
  const loadGraph = useLoadGraph()
  const registerEvents = useRegisterEvents()
  const setSettings = useSetSettings()
  const graphRef = useRef<Graph | null>(null)

  // build + layout once per dataset
  useEffect(() => {
    const graph = new Graph({ multi: false })
    for (const n of data.nodes) {
      graph.addNode(n.id, {
        label: n.name,
        kind: n.kind,
        x: (Math.random() - 0.5) * 100,
        y: (Math.random() - 0.5) * 100,
        size: n.kind === 'person' ? 9 : 5,
        color: NODE_COLORS[n.kind],
      })
    }
    const seen = new Set<string>()
    for (const e of data.edges) {
      const key = e.source + '\x00' + e.target
      if (seen.has(key) || !graph.hasNode(e.source) || !graph.hasNode(e.target)) continue
      seen.add(key)
      graph.addEdge(e.source, e.target, {
        color: e.types?.length
          ? EDGE_COLORS[e.types[0]] || 'rgba(245,245,245,0.3)'
          : 'rgba(245,245,245,0.22)',
        size: e.types?.includes('contradiction') ? 2.2 : 1,
      })
    }
    forceAtlas2.assign(graph, {
      iterations: 60,
      settings: { gravity: 1.5, scalingRatio: 8, barnesHutOptimize: true },
    })
    graphRef.current = graph
    loadGraph(graph)
  }, [data, loadGraph])

  // reducers: hide filtered edges, dim everything but the selected
  // node's neighborhood
  useEffect(() => {
    setSettings({
      nodeReducer: (node, attrs) => {
        const out = { ...attrs }
        if (selected) {
          const neighbors = neighborIds(selected.id, data.edges)
          if (node !== selected.id && !neighbors.has(node)) {
            out.color = 'rgba(245,245,245,0.07)'
            out.label = ''
            out.size = 2
          }
        }
        return out
      },
      edgeReducer: (edge, attrs) => {
        const out = { ...attrs }
        const e = data.edges.find((x) => x.source + '\x00' + x.target === edge)
        if (!e) return out
        if (!edgeMatchesFilter(e, edgeFilter)) out.hidden = true
        if (selected) {
          if (e.source !== selected.id && e.target !== selected.id) out.hidden = true
        }
        return out
      },
    })
  }, [selected, edgeFilter, data, setSettings])

  useEffect(() => {
    registerEvents({
      clickNode: ({ node }) => onNodeClick(node),
    })
  }, [registerEvents, onNodeClick])

  return null
}

export function GraphView({ onNoteSelect }: { onNoteSelect?: (notePath: string) => void }) {
  const { data, loading, error, load } = useGraphData()
  const [activeFilter, setActiveFilter] = useState<Set<string>>(new Set())
  const [selected, setSelected] = useState<GraphNode | null>(null)

  const visibleEdges = useMemo(
    () => (data ? data.edges.filter((e) => edgeMatchesFilter(e, activeFilter)) : []),
    [data, activeFilter],
  )

  const handleNodeClick = useCallback(
    (nodeId: string) => {
      const node = data?.nodes.find((n) => n.id === nodeId)
      if (node) setSelected(node)
    },
    [data],
  )

  const toggleFilter = useCallback((t: string) => {
    setActiveFilter((prev) => {
      const next = new Set(prev)
      if (next.has(t)) next.delete(t)
      else next.add(t)
      return next
    })
  }, [])

  if (loading) {
    return (
      <div className="graph-center" data-testid="graph-loading">
        <div className="graph-center-hint">mapping your mind…</div>
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="graph-center" data-testid="graph-error">
        <div className="graph-center-hint">{error || 'no graph data'}</div>
        <button className="graph-retry" onClick={load}>
          <RefreshCw className="w-3 h-3" /> retry
        </button>
      </div>
    )
  }

  return (
    <div className="graph-view">
      <div className="graph-filters" data-testid="graph-filters">
        <span
          className={cn('gfc', activeFilter.size === 0 && 'on')}
          onClick={() => setActiveFilter(new Set())}
        >
          all
        </span>
        {Object.entries(CONNECTION_TYPE_LABELS).map(([t, label]) => (
          <span
            key={t}
            className={cn('gfc', activeFilter.has(t) && 'on')}
            style={{
              borderColor: activeFilter.has(t) ? EDGE_COLORS[t] : undefined,
              color: activeFilter.has(t) ? EDGE_COLORS[t] : undefined,
            }}
            onClick={() => toggleFilter(t)}
          >
            {label}
          </span>
        ))}
      </div>

      <div className="graph-canvas" data-testid="graph-canvas">
        <SigmaContainer
          style={{ height: '100%', width: '100%', background: 'transparent' }}
          settings={{
            defaultEdgeType: 'line',
            minCameraRatio: 0.2,
            maxCameraRatio: 8,
            labelRenderedSizeThreshold: 7,
            defaultEdgeColor: 'rgba(245,245,245,0.2)',
          }}
        >
          <GraphInner
            data={data}
            edgeFilter={activeFilter}
            selected={selected}
            onNodeClick={handleNodeClick}
          />
        </SigmaContainer>
      </div>

      {selected && (
        <div className="graph-info" data-testid="graph-info">
          <div className="graph-info-kind">{selected.kind}</div>
          <div className="graph-info-name">{selected.name}</div>
          {onNoteSelect && selected.kind === 'note' && (
            <button
              className="graph-info-open"
              onClick={() => onNoteSelect(selected.id)}
              data-testid="graph-open-note"
            >
              open note
            </button>
          )}
          <button className="graph-info-close" onClick={() => setSelected(null)} title="close">
            ×
          </button>
        </div>
      )}

      <div className="graph-legend">
        <span className="gl-item">
          <span className="gl-dot" style={{ background: '#c9933a' }} /> person
        </span>
        <span className="gl-item">
          <span className="gl-dot" style={{ background: '#8a8f98' }} /> note
        </span>
        <span className="gl-item gl-hint">{visibleEdges.length} links</span>
      </div>
    </div>
  )
}
