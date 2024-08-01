// Code generated from Pkl module `org.kdeps.pkl.Resource`. DO NOT EDIT.
package resource

import (
	"github.com/kdeps/kdeps/gen/api"
	"github.com/kdeps/kdeps/gen/env"
	"github.com/kdeps/kdeps/gen/llm"
	"github.com/kdeps/kdeps/gen/settings"
)

type ResourceAction struct {
	Name string `pkl:"name"`

	Exec string `pkl:"exec"`

	Settings *settings.AppSettings `pkl:"settings"`

	Skip *[]string `pkl:"skip"`

	Check *[]string `pkl:"check"`

	Expect *[]string `pkl:"expect"`

	Env *[]*env.ResourceEnv `pkl:"env"`

	Chat *[]*llm.ResourceChat `pkl:"chat"`

	Api *[]*api.ResourceAPI `pkl:"api"`
}
