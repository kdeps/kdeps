// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package validator

import (
	"errors"
	"fmt"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
)

// ExpressionValidator validates custom expression rules.
type ExpressionValidator struct {
	Parser    *expression.Parser
	Evaluator *expression.Evaluator
}

// NewExpressionValidator creates a new expression validator.
func NewExpressionValidator() *ExpressionValidator {
	return &ExpressionValidator{
		Parser: expression.NewParser(),
	}
}

// SetEvaluator sets the evaluator for this validator.
func (v *ExpressionValidator) SetEvaluator(evaluator *expression.Evaluator) {
	v.Evaluator = evaluator
}

// ValidateCustomRules validates custom expression-based rules.
// evaluator and env should be provided by the caller (engine).
func (v *ExpressionValidator) ValidateCustomRules(
	rules []domain.CustomRule,
	evaluator *expression.Evaluator,
	env map[string]interface{},
) error {
	if len(rules) == 0 {
		return nil
	}

	if evaluator == nil {
		return errors.New("evaluator is required for custom rule validation")
	}

	var errors []*domain.ValidationError

	for _, rule := range rules {
		// Get expression string from Expression type
		exprStr := rule.Expr.Raw

		// Evaluate expression using the provided environment
		boolResult, err := evaluator.EvaluateCondition(exprStr, env)
		if err != nil {
			errors = append(errors, &domain.ValidationError{
				Type:    "expression",
				Message: fmt.Sprintf("expression evaluation failed: %v", err),
			})
			continue
		}

		// If false, validation failed
		if !boolResult {
			errors = append(errors, &domain.ValidationError{
				Type:    "custom",
				Message: rule.Message,
			})
		}
	}

	if len(errors) > 0 {
		return &MultipleValidationError{Errors: errors}
	}

	return nil
}
