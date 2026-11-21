// Prototype MCP server for the runs of TASK62.1 item 1. Not product code.
// Speaks line-delimited JSON-RPC on stdio; serves a library built by mklib.py.
package main

import (
	"bufio"
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

const baseUsage = "Quiver serves skills: reusable instructions for specific tasks. At the start of a session Quiver adds a <quiver-skills> block with the id and description of the skills served here, and a <quiver-always-on> block with the always-on skills in full, to the conversation. If there is no <quiver-skills> block, call the list_skills tool of this server before your first reply.\nBefore starting a task, check whether a skill fits. If it does, call the get_skill tool of this server with its id and follow it. The get_skill_file tool loads an extra file of a skill, and search_skills finds skills by words.\n\n"

// Candidate wording of TASK62.8 item 1.
const searchableCue = "Some skills are not listed because the user made them searchable. When the topic of the conversation, not only the last message, calls for something that such a skill could cover (writing for a tool or a channel, working with a service, a file type, a style guide), call search_skills with the topic in a few words, before you start and again when the topic changes. Load a fitting skill with get_skill without asking, because the user chose it. A plain question about the project needs no search."

// Changed wording for the second round of cells (a2, d2): the query names the
// product, tool or channel, and a second search comes from another angle.
const searchableCue2 = "Some skills are not listed because the user made them searchable. When the topic of the conversation, not only the last message, calls for something that such a skill could cover (writing for a tool or a channel, working with a service, a file type, a style guide), call search_skills before you start and again when the topic changes. Write the query as a few words that name the product, tool, service or channel the work is for, and what is being produced. If nothing fits, search once more from another angle (the product alone, or the kind of text). Load a fitting skill with get_skill without asking, because the user chose it. A plain question about the project needs no search."

type indexEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Dir         string `json:"dir"`
}

type rpcReq struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func readLines(path string) map[string]bool {
	out := map[string]bool{}
	if path == "" {
		return out
	}
	b, err := os.ReadFile(path)
	must(err)
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out[l] = true
		}
	}
	return out
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	lib := fs.String("lib", "lib", "library folder with index.json")
	searchable := fs.String("searchable", "", "file with the IDs that search_skills is limited to (empty: the whole library, as today)")
	variant := fs.String("variant", "candidate", "candidate (cue in instructions and tool description) or today")
	cueNo := fs.String("cue", "1", "wording of the cue: 1 or 2")
	engineName := fs.String("engine", "words", "words, today or local")
	logPath := fs.String("log", "", "file for one JSON line per call")
	cutoff := fs.Float64("cutoff", 9.784, "score cutoff of the ranked word search")
	modelDir := fs.String("model", "", "local engine: folder of the in-process model (all-MiniLM-L6-v2)")
	vecFile := fs.String("vecs", "", "local engine: file with the library's embeddings (written by embedlib)")
	vcut := fs.Float64("vcutoff", 0.287, "cosine cutoff of the local engine")
	fs.Parse(args)

	var idx []indexEntry
	readJSON(filepath.Join(*lib, "index.json"), &idx)
	docs := make([]skillDoc, len(idx))
	for i, e := range idx {
		docs[i] = skillDoc{e.ID, e.Name, e.Description}
	}
	bm := newBM25(docs)
	only := readLines(*searchable)
	limited := *searchable != ""

	instructions := baseUsage
	searchDesc := "Finds skills by words in their name, ID or description."
	if *variant == "candidate" {
		cue := searchableCue
		if *cueNo == "2" {
			cue = searchableCue2
		}
		instructions += cue + "\n"
		searchDesc = "Searches the skills that the user made searchable and returns the best matches with their ID and description. " + cue
	}

	var mu sync.Mutex
	var logf *os.File
	if *logPath != "" {
		var err error
		logf, err = os.OpenFile(*logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		must(err)
	}
	logCall := func(v map[string]any) {
		if logf == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		b, _ := json.Marshal(v)
		logf.Write(append(b, '\n'))
	}

	byID := map[string]int{}
	for i, e := range idx {
		byID[e.ID] = i
	}

	var emb Embedder
	var dvec [][]float32
	if *engineName == "local" {
		var err error
		emb, err = LoadAny(*modelDir, true)
		must(err)
		var vf struct{ Docs [][]float32 }
		readJSON(*vecFile, &vf)
		dvec = vf.Docs
		if len(dvec) != len(docs) {
			must(fmt.Errorf("vecs has %d docs, library %d", len(dvec), len(docs)))
		}
	}

	search := func(q string) []int {
		var hs []hit
		if *engineName == "local" {
			var words, vecs []int
			for _, h := range bm.Rank(q) {
				if h.score > *cutoff {
					words = append(words, h.doc)
				}
			}
			qv := emb.EmbedQuery(q)
			var vh []hit
			for i, dv := range dvec {
				if c := float64(dot(qv, dv)); c > *vcut {
					vh = append(vh, hit{i, c})
				}
			}
			slices.SortFunc(vh, func(a, b hit) int { return cmp.Or(cmp.Compare(b.score, a.score), cmp.Compare(a.doc, b.doc)) })
			for _, h := range vh {
				vecs = append(vecs, h.doc)
			}
			score := map[int]float64{}
			for _, list := range [][]int{words, vecs} {
				for r, d := range list {
					score[d] += 1 / float64(60+r+1)
				}
			}
			for d, sc := range score {
				hs = append(hs, hit{d, sc})
			}
			slices.SortFunc(hs, func(a, b hit) int { return cmp.Or(cmp.Compare(b.score, a.score), cmp.Compare(a.doc, b.doc)) })
		} else if *engineName == "today" {
			hs = searchToday(docs, q)
		} else {
			for _, h := range bm.Rank(q) {
				if h.score > *cutoff {
					hs = append(hs, h)
				}
			}
		}
		var out []int
		for _, h := range hs {
			if limited && !only[idx[h.doc].ID] {
				continue
			}
			out = append(out, h.doc)
			if len(out) == 10 {
				break
			}
		}
		return out
	}

	text := func(s string) any {
		return map[string]any{"content": []any{map[string]any{"type": "text", "text": s}}}
	}
	tools := []any{
		map[string]any{"name": "search_skills", "description": searchDesc, "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string", "description": "the topic in a few words"}}, "required": []string{"query"}}},
		map[string]any{"name": "get_skill", "description": "Loads a skill by ID and returns its instructions.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "required": []string{"id"}}},
		map[string]any{"name": "list_skills", "description": "Lists the skills served here with their ID and description.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}},
	}

	handle := func(r rpcReq) (any, bool) {
		switch r.Method {
		case "initialize":
			var p struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			json.Unmarshal(r.Params, &p)
			logCall(map[string]any{"event": "initialize"})
			return map[string]any{"protocolVersion": p.ProtocolVersion, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "quiver", "version": "spike"}, "instructions": instructions}, true
		case "tools/list":
			return map[string]any{"tools": tools}, true
		case "ping":
			return map[string]any{}, true
		case "tools/call":
			var p struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			json.Unmarshal(r.Params, &p)
			switch p.Name {
			case "search_skills":
				q, _ := p.Arguments["query"].(string)
				res := search(q)
				var sb strings.Builder
				var ids []string
				for _, d := range res {
					ids = append(ids, idx[d].ID)
					fmt.Fprintf(&sb, "- %s: %s\n", idx[d].ID, idx[d].Description)
				}
				logCall(map[string]any{"event": "search_skills", "query": q, "results": ids})
				if len(res) == 0 {
					return text("No skills match."), true
				}
				return text(sb.String()), true
			case "get_skill":
				id, _ := p.Arguments["id"].(string)
				i, ok := byID[id]
				logCall(map[string]any{"event": "get_skill", "id": id, "found": ok})
				if !ok {
					return map[string]any{"isError": true, "content": []any{map[string]any{"type": "text", "text": "unknown skill: " + id}}}, true
				}
				b, err := os.ReadFile(filepath.Join(*lib, idx[i].Dir, "SKILL.md"))
				if err != nil {
					return map[string]any{"isError": true, "content": []any{map[string]any{"type": "text", "text": err.Error()}}}, true
				}
				return text(string(b)), true
			case "list_skills":
				logCall(map[string]any{"event": "list_skills"})
				var sb strings.Builder
				for _, e := range idx {
					if !limited || !only[e.ID] {
						fmt.Fprintf(&sb, "- %s: %s\n", e.ID, e.Description)
					}
				}
				return text(sb.String()), true
			}
		}
		return nil, false
	}

	in := bufio.NewReaderSize(os.Stdin, 1<<20)
	out := bufio.NewWriter(os.Stdout)
	for {
		line, err := in.ReadBytes('\n')
		if len(line) > 0 {
			var r rpcReq
			if json.Unmarshal(line, &r) == nil && len(r.ID) > 0 {
				res, ok := handle(r)
				var resp map[string]any
				if ok {
					resp = map[string]any{"jsonrpc": "2.0", "id": r.ID, "result": res}
				} else {
					resp = map[string]any{"jsonrpc": "2.0", "id": r.ID, "error": map[string]any{"code": -32601, "message": "not found"}}
				}
				b, _ := json.Marshal(resp)
				out.Write(append(b, '\n'))
				out.Flush()
			}
		}
		if err != nil {
			return
		}
	}
}

// runEmbedLib embeds the library once so that serve starts at once.
func runEmbedLib(args []string) {
	fs := flag.NewFlagSet("embedlib", flag.ExitOnError)
	lib := fs.String("lib", "lib", "library folder with index.json")
	modelDir := fs.String("model", "", "model folder")
	out := fs.String("out", "vecs.json", "output file")
	fs.Parse(args)
	var idx []indexEntry
	readJSON(filepath.Join(*lib, "index.json"), &idx)
	emb, err := LoadAny(*modelDir, true)
	must(err)
	var vf struct {
		Docs [][]float32 `json:"docs"`
	}
	for _, e := range idx {
		vf.Docs = append(vf.Docs, emb.EmbedDoc(docText(skillDoc{e.ID, e.Name, e.Description})))
	}
	b, _ := json.Marshal(vf)
	must(os.WriteFile(*out, b, 0o644))
}
