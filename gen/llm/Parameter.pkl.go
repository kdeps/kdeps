// Code generated from Pkl module `org.kdeps.pkl.LLM`. DO NOT EDIT.
package llm

type Parameter struct {
	Mirostat int `pkl:"mirostat"`

	MirostatEta float64 `pkl:"mirostat_eta"`

	MirostatTau float64 `pkl:"mirostat_tau"`

	NumCtx int `pkl:"num_ctx"`

	RepeatLastN int `pkl:"repeat_last_n"`

	RepeatPenalty float64 `pkl:"repeat_penalty"`

	Temperature float64 `pkl:"temperature"`

	Seed int `pkl:"seed"`

	TfsZ float64 `pkl:"tfs_z"`

	NumPredict int `pkl:"num_predict"`

	TopK int `pkl:"top_k"`

	TopP float64 `pkl:"top_p"`

	MinP float64 `pkl:"min_p"`
}
