// Code generated from Pkl module `org.kdeps.pkl.Resource`. DO NOT EDIT.
package resource

type AppResource struct {
	Id string `pkl:"id"`

	Name string `pkl:"name"`

	Description string `pkl:"description"`

	Category string `pkl:"category"`

	Requires *[]string `pkl:"requires"`

	Run *[]*ResourceAction `pkl:"run"`
}
