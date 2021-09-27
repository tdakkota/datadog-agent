package obfuscate

// ccObfuscator maintains credit card obfuscation state and processing.
type ccObfuscator struct {
	luhn bool
}

func newCreditCardsObfuscator(useLuhn bool) *ccObfuscator {
	return &ccObfuscator{luhn: useLuhn}
}
