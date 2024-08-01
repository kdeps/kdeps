// Code generated from Pkl module `org.kdeps.pkl.API`. DO NOT EDIT.
package api

type ResourceAPI struct {
	Method string `pkl:"method"`

	Url string `pkl:"url"`

	Data string `pkl:"data"`

	Output *string `pkl:"output"`

	Headers []*ResourceAPIHeaders `pkl:"headers"`
}
