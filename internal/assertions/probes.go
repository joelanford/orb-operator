package assertions

import (
	"fmt"

	"pkg.package-operator.run/boxcutter/probing"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func ProbeForAssertions(assertions []orbv1alpha1.Assertion) (probing.Prober, error) {
	probers := make(probing.And, 0, len(assertions))
	for i, a := range assertions {
		p, err := probeForAssertion(a)
		if err != nil {
			return nil, fmt.Errorf("assertion[%d]: %w", i, err)
		}
		probers = append(probers, p)
	}
	if len(probers) == 0 {
		return nil, nil
	}
	if len(probers) == 1 {
		return probers[0], nil
	}
	return probers, nil
}

func probeForAssertion(a orbv1alpha1.Assertion) (probing.Prober, error) {
	switch {
	case a.ConditionEqual != nil:
		return &probing.ConditionProbe{
			Type:   a.ConditionEqual.Type,
			Status: a.ConditionEqual.Status,
		}, nil
	case a.FieldsEqual != nil:
		return &probing.FieldsEqualProbe{
			FieldA: a.FieldsEqual.FieldA,
			FieldB: a.FieldsEqual.FieldB,
		}, nil
	case a.FieldValue != nil:
		return &probing.FieldValueProbe{
			FieldPath: a.FieldValue.FieldPath,
			Value:     a.FieldValue.Value,
		}, nil
	case a.CELExpression != nil:
		return probing.NewCELProbe(a.CELExpression.Expression, a.CELExpression.Message)
	default:
		return nil, fmt.Errorf("assertion has no recognized type set")
	}
}
