//go:build !js

package llm

// listInstalledOllamaCandidates lists Ollama's pulled tags as installed
// candidates. Split into a build-tagged file because ListOllamaModels
// (service_ollama.go) is itself !js-only -- shelling out to `ollama` makes
// no sense in a wasm build.
func listInstalledOllamaCandidates() []ScoredModelCandidate {
	models := ListOllamaModels()
	out := make([]ScoredModelCandidate, 0, len(models))
	for _, o := range models {
		out = append(out, ScoredModelCandidate{
			Alias: o.Name, Backend: "ollama", Installed: true,
			MatchNames: []string{o.Name},
		})
	}
	return out
}
