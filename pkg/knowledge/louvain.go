// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package knowledge

import "sort"

// maxLouvainPasses bounds the aggregation passes (each one at least halves
// the work in practice).
const maxLouvainPasses = 32

const (
	// gainEpsilon ignores modularity gains lost in float rounding.
	gainEpsilon = 1e-12
	// bothEnds counts an undirected edge's weight once at each end.
	bothEnds = 2
)

// Communities groups the graph's nodes with the Louvain method: each node
// moves to the neighbouring community that most raises modularity, then
// communities become single nodes and the moves repeat, until nothing
// moves. Edges weigh what recall gives them (extracted 1, inferred by
// confidence, tags 0.3). It is deterministic: nodes are visited in id
// order. It sets each node's Community (0 is the largest) and returns the
// partition's modularity.
func Communities(g *Graph) float64 {
	n := len(g.Nodes)
	if n == 0 {
		return 0
	}
	idx := make(map[string]int, n)
	for i, nd := range g.Nodes {
		idx[nd.ID] = i
	}
	adj := make([]map[int]float64, n)
	for i := range adj {
		adj[i] = map[int]float64{}
	}
	for _, e := range g.Edges {
		a, okA := idx[e.From]
		b, okB := idx[e.To]
		if !okA || !okB || a == b {
			continue
		}
		w := edgeWeight(e)
		adj[a][b] += w
		adj[b][a] += w
	}
	member := make([]int, n) // original node -> current community
	for i := range member {
		member[i] = i
	}
	for range maxLouvainPasses {
		comm, moved := localMoves(adj)
		if !moved {
			break
		}
		comm, count := renumber(comm)
		for i := range member {
			member[i] = comm[member[i]]
		}
		adj = aggregate(adj, comm, count)
	}
	final := orderBySize(member, g)
	for i, nd := range g.Nodes {
		nd.Community = final[i]
	}
	return modularity(g, idx)
}

// localMoves is Louvain's first phase on one level's graph.
func localMoves(adj []map[int]float64) ([]int, bool) {
	n := len(adj)
	comm := make([]int, n)
	k := make([]float64, n)
	tot := make([]float64, n)
	var m2 float64
	for i := range adj {
		comm[i] = i
		for _, w := range adj[i] {
			k[i] += w
		}
		tot[i] = k[i]
		m2 += k[i]
	}
	if m2 == 0 {
		return comm, false
	}
	movedAny := false
	for changed := true; changed; {
		changed = false
		for i := range n {
			old := comm[i]
			tot[old] -= k[i]
			best := bestCommunity(i, adj[i], comm, tot, k[i], m2)
			comm[i] = best
			tot[best] += k[i]
			if best != old {
				changed, movedAny = true, true
			}
		}
	}
	return comm, movedAny
}

// bestCommunity is the neighbouring community (or node i's own) whose
// modularity gain for i is highest; ties keep the lower id.
func bestCommunity(i int, row map[int]float64, comm []int, tot []float64, ki, m2 float64) int {
	links := map[int]float64{}
	for j, w := range row {
		if j != i {
			links[comm[j]] += w
		}
	}
	old := comm[i]
	best, bestGain := old, links[old]-tot[old]*ki/m2
	cands := make([]int, 0, len(links))
	for c := range links {
		cands = append(cands, c)
	}
	sort.Ints(cands)
	for _, c := range cands {
		if gain := links[c] - tot[c]*ki/m2; gain > bestGain+gainEpsilon {
			best, bestGain = c, gain
		}
	}
	return best
}

// renumber maps community ids to 0..count-1 in first-seen order.
func renumber(comm []int) ([]int, int) {
	ids := map[int]int{}
	out := make([]int, len(comm))
	for i, c := range comm {
		id, ok := ids[c]
		if !ok {
			id = len(ids)
			ids[c] = id
		}
		out[i] = id
	}
	return out, len(ids)
}

// aggregate makes each community one node; internal weight becomes a
// self-loop, so k and modularity carry over.
func aggregate(adj []map[int]float64, comm []int, count int) []map[int]float64 {
	out := make([]map[int]float64, count)
	for i := range out {
		out[i] = map[int]float64{}
	}
	for i, row := range adj {
		for j, w := range row {
			out[comm[i]][comm[j]] += w
		}
	}
	return out
}

// orderBySize renumbers communities largest first (ties: the community
// holding the smallest node id first).
func orderBySize(member []int, g *Graph) []int {
	size := map[int]int{}
	first := map[int]string{}
	for i, c := range member {
		size[c]++
		if id := g.Nodes[i].ID; first[c] == "" || id < first[c] {
			first[c] = id
		}
	}
	ids := make([]int, 0, len(size))
	for c := range size {
		ids = append(ids, c)
	}
	sort.Slice(ids, func(a, b int) bool {
		if size[ids[a]] != size[ids[b]] {
			return size[ids[a]] > size[ids[b]]
		}
		return first[ids[a]] < first[ids[b]]
	})
	rank := map[int]int{}
	for r, c := range ids {
		rank[c] = r
	}
	out := make([]int, len(member))
	for i, c := range member {
		out[i] = rank[c]
	}
	return out
}

// modularity of the graph's current Community assignment.
func modularity(g *Graph, idx map[string]int) float64 {
	k := make([]float64, len(g.Nodes))
	var m2, in float64
	for _, e := range g.Edges {
		a, okA := idx[e.From]
		b, okB := idx[e.To]
		if !okA || !okB || a == b {
			continue
		}
		w := edgeWeight(e)
		k[a] += w
		k[b] += w
		m2 += bothEnds * w
		if g.Nodes[a].Community == g.Nodes[b].Community {
			in += bothEnds * w
		}
	}
	if m2 == 0 {
		return 0
	}
	tot := map[int]float64{}
	for i, nd := range g.Nodes {
		tot[nd.Community] += k[i]
	}
	q := in / m2
	for _, t := range tot {
		q -= (t / m2) * (t / m2)
	}
	return q
}
