// Code generated from Pkl module `org.kdeps.pkl.Settings`. DO NOT EDIT.
package settings

import (
	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/gen/llm"
)

type AppSettings struct {
	RunTimeout *pkl.Duration `pkl:"runTimeout"`

	InteractiveOnMissingValues bool `pkl:"interactiveOnMissingValues"`

	LlmSettings *llm.LLMSettings `pkl:"llmSettings"`
}
