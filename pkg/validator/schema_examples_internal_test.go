package validator

import "testing"

// A field that a key merely contains gets no borrowed example ("settings is
// required. Example: "/api/users"" came from matching
// settings.apiServer.routes.path); a field path holding a key uses the longest one.
func TestGetFieldExamples_OnlyKeysInsideTheField(t *testing.T) {
	sv := &SchemaValidator{}
	if got := sv.getFieldExamples("settings", "object"); got != `{"key": "value"}` {
		t.Errorf("settings got %s", got)
	}
	if got := sv.getRequiredFieldSuggestion("settings"); got != "This field is required" {
		t.Errorf("required settings got %q", got)
	}
	if got := sv.getFieldExamples("resources.run.chat.model", "string"); got != `"llama3.2:latest"` {
		t.Errorf("nested chat model got %s", got)
	}
	if got := sv.getFieldExamples("metadata.name", "string"); got != `"My Resource"` {
		t.Errorf("metadata.name got %s", got)
	}
}

func TestRequiredFieldPath(t *testing.T) {
	cases := map[[2]string]string{
		{"metadata", "targetActionId is required"}:           "metadata.targetActionId",
		{"(root)", "settings is required"}:                   "settings",
		{"", "settings is required"}:                         "settings",
		{"metadata", "something else"}:                       "metadata",
		{"metadata", "metadata: targetActionId is required"}: "metadata.targetActionId",
		{"settings", "(root): settings is required"}:         "settings",
	}
	for in, want := range cases {
		if got := requiredFieldPath(in[0], in[1]); got != want {
			t.Errorf("requiredFieldPath(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}
