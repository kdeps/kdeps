//go:build js

package llm

// listInstalledOllamaCandidates is a no-op stub for wasm builds: Ollama is a
// local subprocess and irrelevant in a browser, and ListOllamaModels
// (service_ollama.go) is itself !js-only, so there is nothing to list here.
func listInstalledOllamaCandidates() []ScoredModelCandidate {
	return nil
}
