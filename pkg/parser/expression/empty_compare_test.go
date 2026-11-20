package expression_test

import (
	"testing"

	"github.com/kdeps/kdeps/v3/pkg/parser/expression"
)

func TestEvaluateCondition_MissingEqualsEmptyString(t *testing.T) {
	api := createMockAPI()
	_ = api.Set("q", "hello")
	_ = api.Set("empty", "")
	ev := expression.NewEvaluator(api)

	for exprStr, want := range map[string]bool{
		"get('missing') != ''":                false,
		"get('missing') == ''":                true,
		"'' == get('missing')":                true,
		"get('empty') != ''":                  false,
		"get('q') != ''":                      true,
		"get('missing') != nil":               false,
		"get('empty') != nil":                 true,
		"get('q') != '' && len(get('q')) > 3": true,
	} {
		got, err := ev.EvaluateCondition(exprStr, nil)
		if err != nil || got != want {
			t.Errorf("%s = %v (err %v), want %v", exprStr, got, err, want)
		}
	}
}
