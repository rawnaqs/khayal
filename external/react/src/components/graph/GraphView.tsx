import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import Graph from 'graphology'
import forceAtlas2 from 'graphology-layout-forceatlas2'
import EdgeCurveProgram from '@sigma/edge-curve'
import { SigmaContainer, useLoadGraph, useRegisterEvents, useSetSettings, useSigma } from '@react-sigma/core'
import '@react-sigma/core/lib/style.css'
import { createClient, type GraphNode } from '@/lib/api'
import {
  CONNECTION_TYPE_LABELS,
  EDGE_COLORS,
  NODE_COLORS,
  NOTE_TYPE_COLORS,
  degreeMap,
  edgeColor,
  edgeMatchesFilter,
  isHub,
  neighborIds,
  nodeSize,
  withAlpha,
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
  const sigma = useSigma()
  const graphRef = useRef<Graph | null>(null)
  const [hovered, setHovered] = useState<string | null>(null)
  const dragNodeRef = useRef<string | null>(null)

  // build: random spread -> loadGraph -> ForceAtlas2 runs LIVE in small
  // per-frame batches. The graph visibly flows into its shape (the
  // Obsidian feel) instead of popping frozen from a precomputed assign.
  useEffect(() => {
    const graph = new Graph({ multi: false })
    const degrees = degreeMap(data.edges)
    for (const n of data.nodes) {
      const degree = degrees.get(n.id) || 0
      graph.addNode(n.id, {
        label: n.name,
        kind: n.kind,
        hub: isHub(n, degree),
        x: (Math.random() - 0.5) * 40,
        y: (Math.random() - 0.5) * 40,
        size: nodeSize(n, degree),
        // notes pick their hue from the capture type
        color: n.kind === 'person' ? NODE_COLORS.person : NOTE_TYPE_COLORS[n.type || 'text'] || NODE_COLORS.note,
      })
    }
    const seen = new Set<string>()
    for (const e of data.edges) {
      const key = e.source + '\x00' + e.target
      if (seen.has(key) || !graph.hasNode(e.source) || !graph.hasNode(e.target)) continue
      seen.add(key)
      graph.addEdge(e.source, e.target, {
        // whisper-thin, dimmed to ~35%: the web reads as texture, not crayon
        color: withAlpha(edgeColor(e), 0.35),
        size: 0.6,
      })
    }
    graphRef.current = graph
    loadGraph(graph)

    // free-flow physics: a few FA2 iterations per frame, pausing while a
    // node is being dragged so the dragged position sticks.
    // inferSettings scales the force to the graph's node count — the
    // same settings feel right at 50 or 500 nodes.
    let frame = 0
    let raf = 0
    const FA2 = {
      ...forceAtlas2.inferSettings(graph),
      gravity: 1,
      slowDown: 4,
      barnesHutOptimize: true,
      adjustSizes: true,
    }
    const step = () => {
      if (!dragNodeRef.current) {
        // small batch per frame: the graph visibly flows into its shape
        const mapping = forceAtlas2(graph, { iterations: 2, settings: FA2 })
        graph.forEachNode((node) => {
          const pos = mapping[node]
          if (pos) {
            graph.setNodeAttribute(node, 'x', pos.x)
            graph.setNodeAttribute(node, 'y', pos.y)
          }
        })
      }
      // FULL refresh every frame: skipIndexation leaves the hit-test
      // index stale at the initial positions, which made hover/click/
      // drag dead. Hit-test re-indexing at this scale is cheap.
      sigma.refresh()
      if (++frame < 320) raf = requestAnimationFrame(step)
    }
    raf = requestAnimationFrame(step)
    return () => cancelAnimationFrame(raf)
  }, [data, loadGraph, sigma])

  // reducers: hide filtered edges, dim everything but the selected
  // node's neighborhood
  useEffect(() => {
    setSettings({
      nodeReducer: (node, attrs) => {
        const out = { ...attrs }
        const isPerson = graphRef.current?.getNodeAttribute(node, 'kind') === 'person'
        const isHubNode = isPerson || attrs.hub === true
        // hub notes keep labels; minor notes label only in focus
        if (!isHubNode) out.label = ''
        const focusId = selected?.id ?? hovered
        if (focusId) {
          const neighbors = neighborIds(focusId, data.edges)
          if (node !== focusId && !neighbors.has(focusId)) {
            out.color = 'rgba(245,245,245,0.08)'
            out.label = ''
            out.size = 2.5
          } else if (node !== focusId) {
            // focused node's neighbors get a slight boost
            out.size = Math.max(out.size as number, 4)
          }
        } else if (!isPerson && !attrs.hub) {
          out.label = ''
        }
        return out
      },
      edgeReducer: (edge, attrs) => {
        const out = { ...attrs }
        const e = data.edges.find((x) => x.source + '\x00' + x.target === edge)
        if (!e) return out
        if (!edgeMatchesFilter(e, edgeFilter)) out.hidden = true

        const focusId = selected?.id ?? hovered
        if (focusId) {
          if (e.source === focusId || e.target === focusId) {
            // the focused node's connections light up at full strength
            out.color = edgeColor(e)
            out.size = 1.2
          } else {
            out.color = 'rgba(245,245,245,0.05)'
          }
        }
        return out
      },
    })
  }, [selected, edgeFilter, data, setSettings])

  useEffect(() => {
    registerEvents({
      clickNode: ({ node }) => onNodeClick(node),
      enterNode: ({ node }) => setHovered(node),
      leaveNode: () => setHovered(null),
      downNode: ({ node }) => {
        // drag: node follows the pointer, physics pauses for it
        dragNodeRef.current = node
        const container = sigma.getContainer()
        const onMove = (ev: MouseEvent) => {
          const rect = container.getBoundingClientRect()
          const pos = sigma.viewportToGraph({
            x: ev.clientX - rect.left,
            y: ev.clientY - rect.top,
          })
          graphRef.current?.setNodeAttribute(node, 'x', pos.x)
          graphRef.current?.setNodeAttribute(node, 'y', pos.y)
        }
        const onUp = () => {
          dragNodeRef.current = null
          container.removeEventListener('mousemove', onMove)
          container.removeEventListener('mouseup', onUp)
        }
        container.addEventListener('mousemove', onMove)
        container.addEventListener('mouseup', onUp)
      },
    })
  }, [registerEvents, onNodeClick, sigma])

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
            defaultEdgeType: 'curved',
            edgeProgramClasses: { curved: EdgeCurveProgram },
            minCameraRatio: 0.2,
            maxCameraRatio: 8,
            labelRenderedSizeThreshold: 12,
            labelFont: 'IBM Plex Mono, monospace',
            labelColor: { color: 'rgba(245,245,245,0.65)' },
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
