package handler

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// floatToNumeric converts a float64 to pgtype.Numeric.
// Used for dip readings and other decimal values in handler input.
func floatToNumeric(f float64) pgtype.Numeric {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = strings.TrimPrefix(s, "-")
	}

	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}

	intStr := intPart + fracPart
	if intStr == "" {
		intStr = "0"
	}

	i := new(big.Int)
	i.SetString(intStr, 10)
	if neg {
		i.Neg(i)
	}

	return pgtype.Numeric{
		Int:   i,
		Exp:   -int32(len(fracPart)),
		Valid: true,
	}
}

func numericToFloat64(n pgtype.Numeric) (*float64, bool) {
	if !n.Valid || n.Int == nil {
		return nil, false
	}

	value, exact := decimal.NewFromBigInt(n.Int, n.Exp).Float64()
	if !exact {
		return nil, false
	}

	return &value, true
}
