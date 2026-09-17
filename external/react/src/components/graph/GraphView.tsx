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
  buildAdjacency,
  buildEdgeIndex,
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
  onStageClick,
  revealRef,
}: {
  data: RawGraph
  edgeFilter: Set<string>
  selected: GraphNode | null
  onNodeClick: (id: string) => void
  onStageClick: () => void
  revealRef: React.MutableRefObject<number>
}) {
  const loadGraph = useLoadGraph()
  const registerEvents = useRegisterEvents()
  const setSettings = useSetSettings()
  const sigma = useSigma()
  const graphRef = useRef<Graph | null>(null)
  const [hovered, setHovered] = useState<string | null>(null)
  // O(1) lookups for the reducers: resolving an edge or a node's
  // neighbours by scanning the edge list is O(E) / O(N·E) and spikes on
  // hover at scale.
  const edgeIndex = useMemo(() => buildEdgeIndex(data.edges), [data])
  const adjacency = useMemo(() => buildAdjacency(data.edges), [data])
  // Drag state, fully self-owned (sigma's captor has quirks: mousemovebody
  // fires without buttons held, mouseup can bail early). We track the
  // candidate node, the pointer origin, whether real travel happened, and
  // we disable sigma's captor while dragging so the camera stays put.
  const dragCandidateRef = useRef<string | null>(null)
  const dragPointerIdRef = useRef<number | null>(null)
  // which nodes follow a drag, and how strongly (dragged node = 1)
  const dragInfluenceRef = useRef<Map<string, number>>(new Map())
  const dragLastRef = useRef<{ x: number; y: number } | null>(null)
  const touchCountRef = useRef(0)
  const dragActiveRef = useRef(false)
  const physicsPausedRef = useRef(false)
  const downCountRef = useRef(0)
  const clickCountRef = useRef(0)
  // set while we own a gesture (pressed on a node), so sigma's captor
  // handlers know to ignore the click that follows
  const ownGestureRef = useRef(false)
  const onNodeClickRef = useRef(onNodeClick)
  const hitNodeRef = useRef<(x: number, y: number) => string | null>(() => null)

  useEffect(() => {
    onNodeClickRef.current = onNodeClick
  }, [onNodeClick])

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
      if (!dragActiveRef.current && !physicsPausedRef.current) {
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
          const neighbors = adjacency.get(focusId)
          if (node !== focusId && !neighbors?.has(node)) {
            out.color = 'rgba(245,245,245,0.08)'
            out.label = ''
            out.size = 2.5
          }
        }
        return out
      },
      edgeReducer: (edge, attrs) => {
        const out = { ...attrs }
        const e = edgeIndex.get(edge)
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
  }, [selected, hovered, edgeFilter, edgeIndex, adjacency, setSettings, revealRef])

  // Events: hover + empty-space clicks. Node clicks and node drags are
  // owned by the pointer handler below.
  useEffect(() => {
    registerEvents({
      enterNode: ({ node }) => {
        if (!ownGestureRef.current) setHovered(node)
      },
      leaveNode: () => setHovered(null),
      clickNode: ({ node }) => {
        // our pointer handler already handled this gesture
        if (ownGestureRef.current) return
        clickCountRef.current++
        onNodeClick(node)
      },
      // clicking empty space clears the selection
      clickStage: () => {
        if (ownGestureRef.current) return
        onStageClick()
      },
    })
  }, [registerEvents, onNodeClick, onStageClick])

  // Touch policy: ONE finger drags a node and NEVER pans the view; TWO
  // fingers pan/zoom. sigma's touch captor pans on a single finger, so we
  // veto single-finger touchmove before it moves the camera. Two-finger
  // gestures are left to sigma (pan + pinch).
  useEffect(() => {
    const touchCaptor = sigma.getTouchCaptor()
    const onTouchMove = (coords: { touches?: unknown[]; preventSigmaDefault?: () => void }) => {
      if ((coords.touches?.length ?? 0) < 2) coords.preventSigmaDefault?.()
    }
    touchCaptor.on('touchmove', onTouchMove)
    return () => {
      touchCaptor.off('touchmove', onTouchMove)
    }
  }, [sigma])

  // Node drag + selection, fully self-owned.
  //
  // sigma's captor hit test is exact-pixel: a ~3px note dot is ungrabbable,
  // and with physics running the node drifts out from under the cursor, so
  // `downNode` never fires and the camera pans instead. We therefore
  // hit-test ourselves with a tolerance on pointerdown. Clicks on nodes are
  // synthesised on pointerup, and ownGestureRef tells sigma's captor
  // handlers to ignore the trailing click event. On touch, a second finger
  // abandons the node drag so the gesture becomes a view pan.
  useEffect(() => {
    const container = sigma.getContainer()

    const hitNode = (px: number, py: number): string | null => {
      const graph = graphRef.current
      if (!graph) return null
      let best: string | null = null
      let bestDist = Infinity
      graph.forEachNode((id) => {
        const attrs = graph.getNodeAttributes(id)
        const order = (attrs.order as number) ?? 0
        if (revealRef.current - order <= 0) return // still hidden
        const vp = sigma.graphToViewport({ x: attrs.x as number, y: attrs.y as number })
        const dist = Math.hypot(vp.x - px, vp.y - py)
        const size = (attrs.targetSize as number) || 4
        const tolerance = Math.max(size + 8, 14)
        if (dist <= tolerance && dist < bestDist) {
          bestDist = dist
          best = id
        }
      })
      return best
    }
    hitNodeRef.current = hitNode

    // A dragged node pulls its neighbours along, decaying with graph
    // distance, so a connected cluster travels together instead of edges
    // stretching like rubber bands.
    const buildInfluence = (node: string): Map<string, number> => {
      const influence = new Map<string, number>([[node, 1]])
      const graph = graphRef.current
      if (!graph) return influence
      graph.forEachNeighbor(node, (n) => influence.set(n, 0.5))
      graph.forEachNeighbor(node, (n) => {
        graph.forEachNeighbor(n, (n2) => {
          if (!influence.has(n2)) influence.set(n2, 0.15)
        })
      })
      return influence
    }

    const setMouseCaptor = (enabled: boolean) => {
      const captor = sigma.getMouseCaptor()
      if (captor) captor.enabled = enabled
    }

    let start: { x: number; y: number } | null = null
    let travel = 0
    let dragged = false

    const reset = () => {
      dragCandidateRef.current = null
      dragPointerIdRef.current = null
      dragInfluenceRef.current = new Map()
      dragLastRef.current = null
      dragActiveRef.current = false
      start = null
      travel = 0
      dragged = false
    }

    const onDown = (e: PointerEvent) => {
      const isTouch = e.pointerType === 'touch'
      if (isTouch) touchCountRef.current++

      // a second finger means "drag the view", so drop any node drag
      if (isTouch && touchCountRef.current >= 2) {
        if (dragCandidateRef.current) reset()
        return
      }
      if (!isTouch && e.button !== 0) return

      ownGestureRef.current = false
      const rect = container.getBoundingClientRect()
      const px = e.clientX - rect.left
      const py = e.clientY - rect.top
      const node = hitNode(px, py)
      if (!node) return // empty space: mouse pans; one-finger touch does nothing

      ownGestureRef.current = true
      downCountRef.current++
      dragCandidateRef.current = node
      dragPointerIdRef.current = e.pointerId
      dragActiveRef.current = false
      dragInfluenceRef.current = buildInfluence(node)
      dragLastRef.current = sigma.viewportToGraph({ x: px, y: py })
      start = { x: px, y: py }
      travel = 0
      dragged = false
      // Mouse: freeze the camera before sigma sees mousedown (no pan, no
      // stuck state). Touch: leave sigma's touch captor alone — it is
      // needed for two-finger gestures, and single-finger pan is vetoed.
      if (!isTouch) setMouseCaptor(false)
    }

    const onMove = (e: PointerEvent) => {
      const node = dragCandidateRef.current
      if (!node || !start) return
      if (e.pointerId !== dragPointerIdRef.current) return
      const isTouch = e.pointerType === 'touch'
      if (!isTouch && e.buttons === 0) {
        finish(e)
        return
      }
      const rect = container.getBoundingClientRect()
      const px = e.clientX - rect.left
      const py = e.clientY - rect.top
      travel = Math.max(travel, Math.hypot(px - start.x, py - start.y))
      if (!dragged && travel < 3) return
      dragged = true
      dragActiveRef.current = true
      e.preventDefault()
      const pos = sigma.viewportToGraph({ x: px, y: py })
      const last = dragLastRef.current
      const graph = graphRef.current
      if (last && graph) {
        const dx = pos.x - last.x
        const dy = pos.y - last.y
        dragInfluenceRef.current.forEach((factor, id) => {
          if (!graph.hasNode(id)) return
          graph.setNodeAttribute(id, 'x', (graph.getNodeAttribute(id, 'x') as number) + dx * factor)
          graph.setNodeAttribute(id, 'y', (graph.getNodeAttribute(id, 'y') as number) + dy * factor)
        })
      }
      dragLastRef.current = pos
      sigma.refresh()
    }

    const finish = (e: PointerEvent) => {
      const isTouch = e.pointerType === 'touch'
      if (isTouch) touchCountRef.current = Math.max(0, touchCountRef.current - 1)

      const node = dragCandidateRef.current
      if (!node || e.pointerId !== dragPointerIdRef.current) return

      const wasClick = !dragged
      reset()
      if (wasClick) {
        clickCountRef.current++
        onNodeClickRef.current(node)
      }
      if (!isTouch) setMouseCaptor(true)
    }

    container.addEventListener('pointerdown', onDown, true)
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', finish)
    window.addEventListener('pointercancel', finish)
    return () => {
      hitNodeRef.current = () => null
      container.removeEventListener('pointerdown', onDown, true)
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', finish)
      window.removeEventListener('pointercancel', finish)
    }
  }, [sigma, revealRef])

  // DEV-only introspection hook so e2e tests can assert node/camera state
  // (WebGL makes DOM assertions impossible otherwise).
  useEffect(() => {
    if (!import.meta.env.DEV) return
    ;(window as unknown as Record<string, unknown>).__graphDebug = {
      nodeIds: () => graphRef.current?.nodes() ?? [],
      neighbors: (id: string) => graphRef.current?.neighbors(id) ?? [],
      nodeState: (id: string) => {
        const g = graphRef.current
        if (!g || !g.hasNode(id)) return null
        const a = g.getNodeAttributes(id)
        return { x: a.x as number, y: a.y as number, kind: a.kind as string }
      },
      nodeViewport: (id: string) => {
        const g = graphRef.current
        if (!g || !g.hasNode(id)) return null
        const a = g.getNodeAttributes(id)
        return sigma.graphToViewport({ x: a.x as number, y: a.y as number })
      },
      camera: () => sigma.getCamera().getState(),
      canvasRect: () => {
        const el = sigma.getContainer().querySelector('canvas')
        if (!el) return null
        const r = el.getBoundingClientRect()
        return { left: r.left, top: r.top, width: r.width, height: r.height }
      },
      pause: (v: boolean) => {
        physicsPausedRef.current = v
      },
      // tolerant hit test — this is what the pointer handler uses
      hitTest: (x: number, y: number) => hitNodeRef.current(x, y),
      captor: () => {
        const c = sigma.getMouseCaptor()
        return { enabled: c.enabled, isMouseDown: c.isMouseDown, draggedEvents: c.draggedEvents, isMoving: c.isMoving }
      },
      counts: () => ({ downs: downCountRef.current, clicks: clickCountRef.current }),
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
      if (!node) return
      // clicking a note opens it (Obsidian behaviour); people have no
      // note to open, so they surface an info card instead
      if (node.kind === 'note' && onNoteSelect) {
        onNoteSelect(node.id)
        return
      }
      setSelected(node)
    },
    [data, onNoteSelect],
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
            onStageClick={() => setSelected(null)}
            revealRef={revealRef}
          />
        </SigmaContainer>

        {selected && (
          <div className="graph-info" data-testid="graph-info">
            <div className="graph-info-head">
              <span className="graph-info-kind">{selected.kind}</span>
              <button
                className="graph-info-close"
                onPointerDown={(e) => {
                  e.stopPropagation()
                  setSelected(null)
                }}
                onClick={(e) => e.stopPropagation()}
                title="close"
                data-testid="graph-info-close"
              >
                ×
              </button>
            </div>
            <div className="graph-info-name">{selected.name}</div>
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

    </div>
  )
}
