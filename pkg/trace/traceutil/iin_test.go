package traceutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIINValidCardPrefix(t *testing.T) {
	for _, tt := range []struct {
		in         int
		maybe, yes bool
	}{
		// yes
		{1, false, true},
		{4, false, true},
		// maybe
		{2, true, false},
		{3, true, false},
		{5, true, false},
		{6, true, false},
		// no
		{7, false, false},
		{8, false, false},
		{9, false, false},

		// yes
		{34, false, true},
		{37, false, true},
		{39, false, true},
		{51, false, true},
		{55, false, true},
		{62, false, true},
		{65, false, true},
		// maybe
		{30, true, false},
		{63, true, false},
		{22, true, false},
		{27, true, false},
		{69, true, false},
		// no
		{31, false, false},
		{29, false, false},
		{21, false, false},

		// yes
		{300, false, true},
		{305, false, true},
		{644, false, true},
		{649, false, true},
		{309, false, true},
		{636, false, true},
		// maybe
		{352, true, false},
		{358, true, false},
		{501, true, false},
		{601, true, false},
		{222, true, false},
		{272, true, false},
		{500, true, false},
		{509, true, false},
		{560, true, false},
		{589, true, false},
		{600, true, false},
		{699, true, false},

		// yes
		{3528, false, true},
		{3589, false, true},
		{5019, false, true},
		{6011, false, true},
		// maybe
		{2221, true, false},
		{2720, true, false},
		{5000, true, false},
		{5099, true, false},
		{5600, true, false},
		{5899, true, false},
		{6000, true, false},
		{6999, true, false},

		// maybe
		{22210, true, false},
		{27209, true, false},
		{50000, true, false},
		{50999, true, false},
		{56000, true, false},
		{58999, true, false},
		{60000, true, false},
		{69999, true, false},
		// no
		{21000, false, false},
		{55555, false, false},

		// yes
		{222100, false, true},
		{272099, false, true},
		{500000, false, true},
		{509999, false, true},
		{560000, false, true},
		{589999, false, true},
		{600000, false, true},
		{699999, false, true},
		// no
		{551234, false, false},
		{594388, false, false},
		{219899, false, false},
	} {
		t.Run(fmt.Sprintf("%d", tt.in), func(t *testing.T) {
			maybe, yes := validCardPrefix(tt.in)
			assert.Equal(t, maybe, tt.maybe)
			assert.Equal(t, yes, tt.yes)
		})
	}
}

func TestIINIsSensitive(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		for i, valid := range [][]byte{
			[]byte(string("378282246310005")),
			[]byte(string("  378282246310005")),
			[]byte(string("  3782-8224-6310-005 ")),
			[]byte(string("371449635398431")),
			[]byte(string("378734493671000")),
			[]byte(string("5610591081018250")),
			[]byte(string("30569309025904")),
			[]byte(string("38520000023237")),
			[]byte(string("6011 1111 1111 1117")),
			[]byte(string("6011000990139424")),
			[]byte(string(" 3530111333--300000  ")),
			[]byte(string("3566002020360505")),
			[]byte(string("5555555555554444")),
			[]byte(string("5105-1051-0510-5100")),
			[]byte(string(" 4111111111111111")),
			[]byte(string("4012888888881881 ")),
			[]byte(string("422222 2222222")),
			[]byte(string("5019717010103742")),
			[]byte(string("6331101999990016")),
		} {
			assert.True(t, IsSensitive(valid), i)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		for i, invalid := range [][]byte{
			[]byte(string("37828224631000521389798")), // valid but too long
			[]byte(string("37828224631")),             // valid but too short
			[]byte(string("   3782822-4631 ")),
			[]byte(string("3714djkkkksii31")),  // invalid character
			[]byte(string("x371413321323331")), // invalid characters
			[]byte(string("")),
			[]byte(string("7712378231899")),
			[]byte(string("   -  ")),
		} {
			assert.False(t, IsSensitive(invalid), i)
		}
	})
}
