package traceutil

// IsSensitive reports whether b is susceptile to containing PCI
// sensitive data such as credit card information.
func IsSensitive(b []byte) bool {
	//
	// Just credit card numbers for now, based on:
	// • https://baymard.com/checkout-usability/credit-card-patterns
	// • https://www.regular-expressions.info/creditcard.html
	//
	if len(b) == 0 {
		return false
	}
	if b[0] != 0x20 && b[0] != 0x2d && (b[0] < 0x30 || b[0] > 0x39) {
		// fast path: only valid characters are 0-9, space (" ") and dash("-")
		return false
	}
	i := 0               // byte index for traversing b
	num := 0             // holds b[:i] digits as a numeric value (for example []byte{"523"} becomes int(523))
	count := 0           // digit count (excluding white space)
	foundPrefix := false // reports whether we've detected a valid prefix
loop:
	for i < len(b) {
		// We traverse and search b for a valid IIN credit card prefix based
		// on the digits found, ignoring spaces and dashes.
		// Source: https://www.regular-expressions.info/creditcard.html
		switch b[i] {
		case 0x20, 0x2d:
			// ignore space (' ') and dash ('-')
			i++
			continue loop
		}
		if b[i] < 0x30 || b[i] > 0x39 {
			// not a 0 to 9 digit; can not be a credit card number; abort
			return false
		}
		if !foundPrefix {
			// we have not yet found a valid prefix so we convert the digits
			// that we have so far into a numeric value:
			num = num*10 + (int(b[i]) - 0x30)
			maybe, yes := validCardPrefix(num)
			if yes {
				// we've found a valid prefix; continue counting
				foundPrefix = true
			} else if !maybe {
				// this is not a valid prefix and we should not continue looking
				return false
			}
		}
		count++
		if count > 16 {
			// too many digits
			return false
		}
		i++
	}
	if count < 12 {
		// too few digits
		return false
	}
	return foundPrefix
}

// validCardPrefix validates whether b is a valid card prefix. It is expected
// to be strictly numeric within byte range 0x30-0x39. Maybe returns true if
// the prefix could be an IIN once more digits are revealed and yes reports
// whether b is a fully valid IIN.
//
// If yes is false and maybe is false, there is no reason to continue searching.
func validCardPrefix(n int) (maybe, yes bool) {
	// Validates IIN prefix possibilities
	// Source: https://www.regular-expressions.info/creditcard.html
	if n > 699999 {
		// too long for any known prefix
		return false, false
	}
	if n < 10 {
		switch n {
		case 1, 4:
			// 1 & 4 are valid IIN
			return false, true
		case 2, 3, 5, 6:
			// 2, 3, 5, 6 could be the start of valid IIN
			return true, false
		default:
			// invalid IIN
			return false, false
		}
	}
	if n < 100 {
		if (n >= 34 && n <= 39) ||
			(n >= 51 && n <= 55) ||
			n == 62 ||
			n == 65 {
			// 34-39, 51-55, 62, 65 are valid IIN
			return false, true
		}
		if n == 30 || n == 63 || n == 64 || n == 35 || n == 50 || n == 60 ||
			(n >= 22 && n <= 27) || (n >= 56 && n <= 58) || (n >= 60 && n <= 69) {
			// 30, 63, 64, 35, 50, 60, 22-27, 56-58, 60-69 may end up as valid IIN
			return true, false
		}
	}
	if n < 1000 {
		if (n >= 300 && n <= 305) ||
			(n >= 644 && n <= 649) ||
			n == 309 ||
			n == 636 {
			// 300‑305, 309, 636, 644‑649 are valid IIN
			return false, true
		}
		if (n >= 352 && n <= 358) || n == 501 || n == 601 ||
			(n >= 222 && n <= 272) || (n >= 500 && n <= 509) ||
			(n >= 560 && n <= 589) || (n >= 600 && n <= 699) {
			// 352-358, 501, 601, 222-272, 500-509, 560-589, 600-699 may be IIN
			return true, false
		}
	}
	if n < 10000 {
		if (n >= 3528 && n <= 3589) ||
			n == 5019 ||
			n == 6011 {
			// 3528‑3589, 5019, 6011 are valid IINs
			return false, true
		}
		if (n >= 2221 && n <= 2720) || (n >= 5000 && n <= 5099) ||
			(n >= 5600 && n <= 5899) || (n >= 6000 && n <= 6999) {
			return true, false
		}
	}
	if n < 100000 {
		if (n >= 22210 && n <= 27209) ||
			(n >= 50000 && n <= 50999) ||
			(n >= 56000 && n <= 58999) ||
			(n >= 60000 && n <= 69999) {
			// maybe a 6-digit IIN
			return true, false
		}
	}
	if n < 1000000 {
		if (n >= 222100 && n <= 272099) ||
			(n >= 500000 && n <= 509999) ||
			(n >= 560000 && n <= 589999) ||
			(n >= 600000 && n <= 699999) {
			// 222100‑272099, 500000‑509999, 560000‑589999, 600000‑699999 are valid IIN
			return false, true
		}
	}
	// unknown IIN
	return false, false
}
