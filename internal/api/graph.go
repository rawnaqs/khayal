package api

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rawnaqs/khayal/internal/queue"
)

// GraphNode is a vertex in the connection graph: a note or a person.
type GraphNode struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"` // "note" | "person"
	Name    string `json:"name"`
	Type    string `json:"type,omitempty"`    // note type: text/image/article/pdf
	Created string `json:"created,omitempty"` // RFC3339, drives age-staggered reveal
}

// GraphEdge is a directed relationship: note→note (connections, with the
// detector types that produced it) or person→note (mentions).
type GraphEdge struct {
	Source string   `json:"source"`
	Target string   `json:"target"`
	Types  []string `json:"types,omitempty"`
}

// Graph is the full connection web, sized for the personal-vault use case.
type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// graphMaxNodes caps rendering load: personal vaults are small, but a
// runaway index should degrade instead of freezing the client.
const graphMaxNodes = 500

// graphHandler serves the whole connection web: notes, people, and every
// relationship between them. Feed for a future visualization.
func (s *Server) graphHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	g := Graph{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
	}

	// --- note-note edges from the connections results -------------------
	type storedConn struct {
		NotePath string `json:"note_path"`
		Type     string `json:"type"`
	}
	connBySource := map[string][]storedConn{}
	results, err := s.queue.AllConnectionResults(ctx)
	if err == nil {
		for _, jr := range results {
			var payload struct {
				Connections []storedConn `json:"connections"`
			}
			if json.Unmarshal([]byte(jr.Result), &payload) != nil {
				continue
			}
			connBySource[jr.NotePath] = append(connBySource[jr.NotePath], payload.Connections...)
		}
	}

	// --- note + person nodes from the entity table ---
	entities, err := s.queue.AllPersonEntities(ctx)
	if err != nil {
		entities = nil
	}

	nodeSeen := map[string]bool{}
	noteMeta := map[string]queue.NoteGraphMeta{}
	if metaList, err := s.queue.GraphNoteMeta(ctx); err == nil {
		for _, m := range metaList {
			noteMeta[m.NotePath] = m
		}
	}
	addNoteNode := func(path string) {
		if path == "" || nodeSeen[path] {
			return
		}
		nodeSeen[path] = true
		m, ok := noteMeta[path]
		if !ok {
			m = queue.NoteGraphMeta{NotePath: path, Type: "text", Title: strings.TrimSuffix(filepath.Base(path), ".md")}
		}
		g.Nodes = append(g.Nodes, GraphNode{ID: path, Kind: "note", Name: m.Title, Type: m.Type, Created: m.Created})
	}
	personAdded := map[string]bool{}
	addPersonNode := func(name string) {
		id := "person:" + strings.ToLower(name)
		if personAdded[id] {
			return
		}
		personAdded[id] = true
		g.Nodes = append(g.Nodes, GraphNode{ID: id, Kind: "person", Name: name})
	}

	for _, e := range entities {
		addNoteNode(e.NotePath)
		addPersonNode(e.Value)
		g.Edges = append(g.Edges, GraphEdge{Source: "person:" + strings.ToLower(e.Value), Target: e.NotePath})
	}
	for _, conns := range connBySource {
		for _, c := range conns {
			addNoteNode(c.NotePath)
		}
	}
	for src, conns := range connBySource {
		addNoteNode(src)
		for _, c := range conns {
			// merge duplicate edges (same pair, multiple types)
			merged := false
			for i := range g.Edges {
				if g.Edges[i].Source == src && g.Edges[i].Target == c.NotePath {
					g.Edges[i].Types = appendUnique(g.Edges[i].Types, c.Type)
					merged = true
					break
				}
			}
			if !merged {
				g.Edges = append(g.Edges, GraphEdge{Source: src, Target: c.NotePath, Types: []string{c.Type}})
			}
		}
	}

	// cap nodes: keep the most recent notes when over budget
	if len(g.Nodes) > graphMaxNodes {
		notes, people := 0, 0
		var kept []GraphNode
		for _, n := range g.Nodes {
			if n.Kind == "person" {
				people++
				if people <= graphMaxNodes/2 {
					kept = append(kept, n)
				}
			} else {
				notes++
				if notes <= graphMaxNodes/2 {
					kept = append(kept, n)
				}
			}
		}
		sort.SliceStable(kept, func(i, j int) bool { return kept[i].ID < kept[j].ID })
		g.Nodes = kept[:min(len(kept), graphMaxNodes)]
	}
	WriteJSON(w, http.StatusOK, g)
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}
