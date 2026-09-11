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
  revealProgress,
  withAlpha,
} from '@/lib/graphModel'
import { useVaultLock } from '@/hooks/useVaultLock'
import { cn } from '@/lib/utils'

type RawGraph = { nodes: GraphNode[]; edges: { source: string; target: string; types?: string[] }[] }
type RawEdge = RawGraph['edges'][number]

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

// Reveal order: notes oldest -> newest, people last (they are derived).
function revealOrderedNodes(nodes: GraphNode[]): GraphNode[] {
  const notes = nodes
    .filter((n) => n.kind === 'note')
    .sort((a, b) => (a.created || '9999') < (b.created || '9999') ? -1 : 1)
  const people = nodes.filter((n) => n.kind === 'person')
  return [...notes, ...people]
}

// GraphInner lives inside SigmaContainer (needs the sigma context).
function GraphInner({
  data,
  edgeFilter,
  selected,
  onNodeClick,
  revealRef,
}: {
  data: RawGraph
  edgeFilter: Set<string>
  selected: GraphNode | null
  onNodeClick: (id: string) => void
  revealRef: React.MutableRefObject<number>
}) {
  const loadGraph = useLoadGraph()
  const registerEvents = useRegisterEvents()
  const setSettings = useSetSettings()
  const sigma = useSigma()
  const graphRef = useRef<Graph | null>(null)
  const [hovered, setHovered] = useState<string | null>(null)
  const dragNodeRef = useRef<string | null>(null)

  // Build: deterministic edge keys (so reducers can match edges by key),
  // age-ordered nodes, all hidden until the reveal sweeps them in.
  useEffect(() => {
    const graph = new Graph({ multi: false })
    const ordered = revealOrderedNodes(data.nodes)
    const degrees = degreeMap(data.edges)
    revealRef.current = 0

    ordered.forEach((n, index) => {
      const degree = degrees.get(n.id) || 0
      const size = nodeSize(n, degree)
      graph.addNode(n.id, {
        label: n.name,
        kind: n.kind,
        hub: isHub(n, degree),
        order: index,
        targetSize: size,
        size: 0,
        x: (Math.random() - 0.5) * 40,
        y: (Math.random() - 0.5) * 40,
        color:
          n.kind === 'person'
            ? NODE_COLORS.person
            : NOTE_TYPE_COLORS[n.type || 'text'] || NODE_COLORS.note,
      })
    })

    const seen = new Set<string>()
    for (const e of data.edges) {
      const key = e.source + '\u0000' + e.target
      if (seen.has(key) || !graph.hasNode(e.source) || !graph.hasNode(e.target)) continue
      seen.add(key)
      // explicit key: reducers receive this key and can map back to the
      // edge's detector types (auto-generated keys broke filters)
      graph.addEdgeWithKey(key, e.source, e.target, {
        nodeColor: edgeColor(e),
        color: withAlpha(edgeColor(e), 0.35),
        size: 0.6,
      })
    }
    graphRef.current = graph
    loadGraph(graph)

    // Physics + reveal: one gentle FA2 iteration per frame, forever;
    // the reveal counter sweeps nodes in oldest-first.
    const FA2 = {
      ...forceAtlas2.inferSettings(graph),
      gravity: 1,
      barnesHutOptimize: true,
      adjustSizes: true,
    }
    let frame = 0
    const rafRef = { current: 0 }
    const step = () => {
      frame++
      // reveal ~all nodes over ~1.5s
      revealRef.current = revealProgress(revealRef.current, ordered.length)
      if (!dragNodeRef.current) {
        const slowDown = Math.min(2 + frame * 0.04, 10)
        const iterations = frame < 30 ? 4 : frame < 90 ? 2 : 1
        const mapping = forceAtlas2(graph, { iterations, settings: { ...FA2, slowDown } })
        graph.forEachNode((node) => {
          const pos = mapping[node]
          if (pos) {
            graph.setNodeAttribute(node, 'x', pos.x)
            graph.setNodeAttribute(node, 'y', pos.y)
          }
        })
      }
      // FULL refresh every frame keeps the hit-test index live.
      sigma.refresh()
      rafRef.current = requestAnimationFrame(step)
    }
    rafRef.current = requestAnimationFrame(step)
    return () => cancelAnimationFrame(rafRef.current)
  }, [data, loadGraph, sigma, revealRef])

  // Reducers: reveal staging + filters + focus dimming.
  useEffect(() => {
    setSettings({
      nodeReducer: (node, attrs) => {
        const out = { ...attrs }
        const order = (attrs.order as number) ?? 0
        const reveal = revealRef.current - order
        if (reveal <= 0) {
          out.hidden = true
          return out
        }
        // ease in over ~8 nodes of reveal budget
        const t = Math.min(reveal / 8, 1)
        out.hidden = false
        out.size = ((attrs.targetSize as number) || 4) * (0.3 + 0.7 * t)

        const isPerson = attrs.kind === 'person'
        const isHubNode = isPerson || attrs.hub === true
        if (!isHubNode) out.label = ''
        const focusId = selected?.id ?? hovered
        if (focusId) {
          const neighbors = neighborIds(focusId, data.edges)
          if (node !== focusId && !neighbors.has(node)) {
            out.color = 'rgba(245,245,245,0.08)'
            out.label = ''
            out.size = 2.5
          }
        }
        return out
      },
      edgeReducer: (edge, attrs) => {
        const out = { ...attrs }
        const e = data.edges.find((x) => x.source + '\u0000' + x.target === edge)
        if (!e) return out

        // hide until both endpoints have been revealed
        const orderOf = (id: string) => {
          const o = graphRef.current?.getNodeAttribute(id, 'order')
          return typeof o === 'number' ? o : 0
        }
        if (revealRef.current - orderOf(e.source) <= 0 || revealRef.current - orderOf(e.target) <= 0) {
          out.hidden = true
          return out
        }

        if (!edgeMatchesFilter(e, edgeFilter)) out.hidden = true

        const focusId = selected?.id ?? hovered
        if (focusId) {
          if (e.source === focusId || e.target === focusId) {
            out.color = edgeColor(e)
            out.size = 1.2
          } else {
            out.color = 'rgba(245,245,245,0.05)'
          }
        }
        return out
      },
    })
  }, [selected, hovered, edgeFilter, data, setSettings, revealRef])

  // Drag: sigma's own captor pattern — preventSigmaDefault() vetoes the
  // camera pan while a node is held; empty-space drags still pan.
  useEffect(() => {
    registerEvents({
      clickNode: ({ node }) => onNodeClick(node),
      enterNode: ({ node }) => setHovered(node),
      leaveNode: () => setHovered(null),
      downNode: ({ node }) => {
        dragNodeRef.current = node
      },
    })
  }, [registerEvents, onNodeClick])

  useEffect(() => {
    const captor = sigma.getMouseCaptor()
    if (!captor) return

    const onMove = (e: {
      x: number
      y: number
      preventSigmaDefault?: () => void
      original?: { preventDefault?: () => void; stopPropagation?: () => void }
    }) => {
      const node = dragNodeRef.current
      if (!node) return
      e.preventSigmaDefault?.()
      e.original?.preventDefault?.()
      e.original?.stopPropagation?.()
      const pos = sigma.viewportToGraph({ x: e.x, y: e.y })
      graphRef.current?.setNodeAttribute(node, 'x', pos.x)
      graphRef.current?.setNodeAttribute(node, 'y', pos.y)
      sigma.refresh()
    }
    const onUp = () => {
      dragNodeRef.current = null
    }
    // disable autoscale on first interaction so the camera doesn't fight
    // the dragged node (official sigma drag example does the same)
    const onDown = () => {
      if (!sigma.getCustomBBox()) sigma.setCustomBBox(sigma.getBBox())
    }

    captor.on('mousemovebody', onMove)
    captor.on('mouseup', onUp)
    captor.on('mousedown', onDown)
    return () => {
      captor.off('mousemovebody', onMove)
      captor.off('mouseup', onUp)
      captor.off('mousedown', onDown)
    }
  }, [sigma])

  return null
}

export function GraphView({ onNoteSelect }: { onNoteSelect?: (notePath: string) => void }) {
  const { data, loading, error, load } = useGraphData()
  const [activeFilter, setActiveFilter] = useState<Set<string>>(new Set())
  const [selected, setSelected] = useState<GraphNode | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const revealRef = useRef(0)

  // filter edges client-side for the link counter
  const visibleEdges = useMemo(() => {
    if (!data) return [] as RawEdge[]
    return data.edges.filter((e) => edgeMatchesFilter(e, activeFilter))
  }, [data, activeFilter])

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

  const handleReload = useCallback(() => {
    setSelected(null)
    setActiveFilter(new Set())
    setReloadKey((k) => k + 1)
    load()
  }, [load])

  if (loading && !data) {
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
        <button
          className="graph-reload"
          onClick={handleReload}
          title="rebuild the graph"
          data-testid="graph-reload"
        >
          <RefreshCw className={cn('w-3 h-3', loading && 'animate-spin')} />
        </button>
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
            key={reloadKey}
            data={data}
            edgeFilter={activeFilter}
            selected={selected}
            onNodeClick={handleNodeClick}
            revealRef={revealRef}
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
          <span className="gl-dot" style={{ background: '#8a93a6' }} /> note
        </span>
        <span className="gl-item gl-hint">{visibleEdges.length} links</span>
      </div>
    </div>
  )
}
