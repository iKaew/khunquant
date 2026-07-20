package settrade

import "testing"

func TestRoundToTickSize_PreservesExactValidTick(t *testing.T) {
	const requestedPrice = 3.76

	got := roundToTickSize(requestedPrice)

	if got != requestedPrice {
		t.Fatalf(
			"roundToTickSize(%0.2f) = %.20f (%0.2f); want the exact valid tick price to remain %0.2f",
			requestedPrice,
			got,
			got,
			requestedPrice,
		)
	}
}
