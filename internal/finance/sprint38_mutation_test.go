package finance

import (
	"testing"

	"github.com/rvbernucci/signalforge/internal/numeric"
)

func TestSprint38HighRiskFormulaMutationMatrixFailsClosed(t *testing.T) {
	forecast := []numeric.Decimal{
		numeric.MustDecimal("10"),
		numeric.MustDecimal("11"),
		numeric.MustDecimal("12"),
	}
	tests := map[string]func() error{
		"empty forecast": func() error {
			_, err := FCFFDCF(nil, numeric.MustDecimal("0.10"), numeric.MustDecimal("0.03"))
			return err
		},
		"discount equals terminal growth": func() error {
			_, err := FCFFDCF(forecast, numeric.MustDecimal("0.03"), numeric.MustDecimal("0.03"))
			return err
		},
		"discount below terminal growth": func() error {
			_, err := FCFFDCF(forecast, numeric.MustDecimal("0.02"), numeric.MustDecimal("0.03"))
			return err
		},
		"discount at negative one": func() error {
			_, err := FCFFDCF(forecast, numeric.MustDecimal("-1"), numeric.MustDecimal("-1.1"))
			return err
		},
		"empty discount axis": func() error {
			_, err := DCFGrid(nil, nil, []numeric.Decimal{numeric.MustDecimal("0.03")})
			return err
		},
		"empty terminal growth axis": func() error {
			_, err := DCFGrid(nil, []numeric.Decimal{numeric.MustDecimal("0.10")}, nil)
			return err
		},
		"invalid decimal": func() error {
			_, err := numeric.ParseDecimal("not-a-number")
			return err
		},
	}
	for name, run := range tests {
		t.Run(name, func(t *testing.T) {
			if err := run(); err == nil {
				t.Fatal("high-risk formula mutation did not fail closed")
			}
		})
	}
}
