package utils_test

import (
	"testing"

	. "github.com/kdeps/kdeps/pkg/utils"
)

func TestShouldSkip_Internal(t *testing.T) {
	cases := [][]interface{}{
		{true, false},
		{"TRUE", false},
		{false, "TrUe"},
	}
	for _, c := range cases {
		if !ShouldSkip(&c) {
			t.Errorf("ShouldSkip failed for %v", c)
		}
	}
	noSkip := []interface{}{false, "no"}
	if ShouldSkip(&noSkip) {
		t.Error("ShouldSkip returned true for all-false case")
	}
}

func TestAllConditionsMet_Internal(t *testing.T) {
	trueSet := []interface{}{true, "TRUE"}
	if !AllConditionsMet(&trueSet) {
		t.Error("AllConditionsMet failed for all-true case")
	}
	falseSet := []interface{}{true, "false"}
	if AllConditionsMet(&falseSet) {
		t.Error("AllConditionsMet returned true for mixed case")
	}
	unsupported := []interface{}{123}
	if AllConditionsMet(&unsupported) {
		t.Error("AllConditionsMet returned true for unsupported type")
	}
}
