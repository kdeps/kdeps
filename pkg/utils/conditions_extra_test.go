package utils

import "testing"

func TestAllConditionsMetExtra(t *testing.T) {
	t.Run("all true bools", func(t *testing.T) {
		conds := []interface{}{true, true, true}
		if !AllConditionsMet(&conds) {
			t.Fatalf("expected all conditions met")
		}
	})

	t.Run("one false bool", func(t *testing.T) {
		conds := []interface{}{true, false, true}
		if AllConditionsMet(&conds) {
			t.Fatalf("expected not all conditions met")
		}
	})

	t.Run("string true values", func(t *testing.T) {
		conds := []interface{}{"TRUE", "true", "TrUe"}
		if !AllConditionsMet(&conds) {
			t.Fatalf("expected all string conditions met")
		}
	})

	t.Run("string non true", func(t *testing.T) {
		conds := []interface{}{"true", "false"}
		if AllConditionsMet(&conds) {
			t.Fatalf("expected not all conditions met when one string is false")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		conds := []interface{}{true, 123}
		if AllConditionsMet(&conds) {
			t.Fatalf("expected not all conditions met with unsupported type")
		}
	})
}
