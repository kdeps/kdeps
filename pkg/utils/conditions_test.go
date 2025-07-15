package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/pkg/utils"
)

func TestShouldSkip(t *testing.T) {
	// Test skip condition
	val1 := []interface{}{"skip"}
	assert.True(t, utils.ShouldSkip(&val1))
	val2 := []interface{}{"SKIP"}
	assert.True(t, utils.ShouldSkip(&val2))
	val3 := []interface{}{"Skip"}
	assert.True(t, utils.ShouldSkip(&val3))

	// Test non-skip conditions
	val4 := []interface{}{"continue"}
	assert.False(t, utils.ShouldSkip(&val4))
	val5 := []interface{}{""}
	assert.False(t, utils.ShouldSkip(&val5))
	val6 := []interface{}{"other"}
	assert.False(t, utils.ShouldSkip(&val6))
}

func TestAllConditionsMet(t *testing.T) {
	// Test all conditions met
	vals1 := []interface{}{true, true}
	assert.True(t, utils.AllConditionsMet(&vals1))
	vals2 := []interface{}{}
	assert.True(t, utils.AllConditionsMet(&vals2))
	vals3 := []interface{}{true}
	assert.True(t, utils.AllConditionsMet(&vals3))

	// Test conditions not met
	vals4 := []interface{}{true, false}
	assert.False(t, utils.AllConditionsMet(&vals4))
	vals5 := []interface{}{false}
	assert.False(t, utils.AllConditionsMet(&vals5))
}

func TestShouldSkipAndAllConditionsMet(t *testing.T) {
	cases := []struct {
		name       string
		input      []interface{}
		wantSkip   bool
		wantAllMet bool
	}{
		{"all bool true", []interface{}{true, true}, true, true},
		{"mixed true string", []interface{}{false, "true"}, true, false},
		{"all false", []interface{}{false, false}, false, false},
		{"all string true", []interface{}{"true", "true"}, true, true},
		{"mixed false", []interface{}{true, "false"}, true, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := utils.ShouldSkip(&tc.input); got != tc.wantSkip {
				t.Fatalf("ShouldSkip(%v) = %v, want %v", tc.input, got, tc.wantSkip)
			}
			if got := utils.AllConditionsMet(&tc.input); got != tc.wantAllMet {
				t.Fatalf("AllConditionsMet(%v) = %v, want %v", tc.input, got, tc.wantAllMet)
			}
		})
	}
}

func TestAllConditionsMetExtra(t *testing.T) {
	t.Run("all true bools", func(t *testing.T) {
		conds := []interface{}{true, true, true}
		if !utils.AllConditionsMet(&conds) {
			t.Fatalf("expected all conditions met")
		}
	})

	t.Run("one false bool", func(t *testing.T) {
		conds := []interface{}{true, false, true}
		if utils.AllConditionsMet(&conds) {
			t.Fatalf("expected not all conditions met")
		}
	})

	t.Run("string true values", func(t *testing.T) {
		conds := []interface{}{"TRUE", "true", "TrUe"}
		if !utils.AllConditionsMet(&conds) {
			t.Fatalf("expected all string conditions met")
		}
	})

	t.Run("string non true", func(t *testing.T) {
		conds := []interface{}{"true", "false"}
		if utils.AllConditionsMet(&conds) {
			t.Fatalf("expected not all conditions met when one string is false")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		conds := []interface{}{true, 123}
		if utils.AllConditionsMet(&conds) {
			t.Fatalf("expected not all conditions met with unsupported type")
		}
	})
}
