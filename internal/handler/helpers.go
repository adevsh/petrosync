package handler

import (
	"math/big"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// floatToNumeric converts a float64 to pgtype.Numeric.
// Used for dip readings and other decimal values in handler input.
func floatToNumeric(f float64) pgtype.Numeric {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	n := new(big.Rat)
	n.SetString(s)
	return pgtype.Numeric{
		Int:   n.Num(),
		Exp:   -int32(len(n.Denom().Text(10)) - 1),
		Valid: true,
	}
}
