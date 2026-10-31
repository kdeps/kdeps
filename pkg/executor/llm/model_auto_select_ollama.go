package llm

import "context"

// listInstalledOllamaCandidates lists Ollama's pulled tags as installed
// candidates.
func listInstalledOllamaCandidates(ctx context.Context) []ScoredModelCandidate {
	models := ListOllamaModels(ctx)
	out := make([]ScoredModelCandidate, 0, len(models))
	for _, o := range models {
		out = append(out, ScoredModelCandidate{
			Alias: o.Name, Backend: "ollama", Installed: true,
			MatchNames: []string{o.Name},
		})
	}
	return out
}
