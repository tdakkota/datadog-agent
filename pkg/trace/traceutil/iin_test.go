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
			[]byte(string(" 4242-4242-4242-4242 ")),
			[]byte(string("4242-4242-4242-4242 ")),
			[]byte(string("4242-4242-4242-4242  ")),
			[]byte(string("4000056655665556")),
			[]byte(string("5555555555554444")),
			[]byte(string("2223003122003222")),
			[]byte(string("5200828282828210")),
			[]byte(string("5105105105105100")),
			[]byte(string("378282246310005")),
			[]byte(string("371449635398431")),
			[]byte(string("6011111111111117")),
			[]byte(string("6011000990139424")),
			[]byte(string("3056930009020004")),
			[]byte(string("3566002020360505")),
			[]byte(string("620000000000000")),
			[]byte(string("2222 4053 4324 8877")),
			[]byte(string("2222 9909 0525 7051")),
			[]byte(string("2223 0076 4872 6984")),
			[]byte(string("2223 5771 2001 7656")),
			[]byte(string("5105 1051 0510 5100")),
			[]byte(string("5111 0100 3017 5156")),
			[]byte(string("5185 5408 1000 0019")),
			[]byte(string("5200 8282 8282 8210")),
			[]byte(string("5204 2300 8000 0017")),
			[]byte(string("5204 7400 0990 0014")),
			[]byte(string("5420 9238 7872 4339")),
			[]byte(string("5455 3307 6000 0018")),
			[]byte(string("5506 9004 9000 0436")),
			[]byte(string("5506 9004 9000 0444")),
			[]byte(string("5506 9005 1000 0234")),
			[]byte(string("5506 9208 0924 3667")),
			[]byte(string("5506 9224 0063 4930")),
			[]byte(string("5506 9274 2731 7625")),
			[]byte(string("5553 0422 4198 4105")),
			[]byte(string("5555 5537 5304 8194")),
			[]byte(string("5555 5555 5555 4444")),
			[]byte(string("4012 8888 8888 1881")),
			[]byte(string("4111 1111 1111 1111")),
			[]byte(string("6011 0009 9013 9424")),
			[]byte(string("6011 1111 1111 1117")),
			[]byte(string("3714 496353 98431")),
			[]byte(string("3782 822463 10005")),
			[]byte(string("3056 9309 0259 04")),
			[]byte(string("3852 0000 0232 37")),
			[]byte(string("3530 1113 3330 0000")),
			[]byte(string("3566 0020 2036 0505")),
			[]byte(string("3700 0000 0000 002")),
			[]byte(string("3700 0000 0100 018")),
			[]byte(string("6703 4444 4444 4449")),
			[]byte(string("4871 0499 9999 9910")),
			[]byte(string("4035 5010 0000 0008")),
			[]byte(string("4360 0000 0100 0005")),
			[]byte(string("6243 0300 0000 0001")),
			[]byte(string("5019 5555 4444 5555")),
			[]byte(string("3607 0500 0010 20")),
			[]byte(string("6011 6011 6011 6611")),
			[]byte(string("6445 6445 6445 6445")),
			[]byte(string("5066 9911 1111 1118")),
			[]byte(string("6062 8288 8866 6688")),
			[]byte(string("3569 9900 1009 5841")),
			[]byte(string("6771 7980 2100 0008")),
			[]byte(string("2222 4000 7000 0005")),
			[]byte(string("5555 3412 4444 1115")),
			[]byte(string("5577 0000 5577 0004")),
			[]byte(string("5555 4444 3333 1111")),
			[]byte(string("2222 4107 4036 0010")),
			[]byte(string("5555 5555 5555 4444")),
			[]byte(string("2222 4107 0000 0002")),
			[]byte(string("2222 4000 1000 0008")),
			[]byte(string("2223 0000 4841 0010")),
			[]byte(string("2222 4000 6000 0007")),
			[]byte(string("2223 5204 4356 0010")),
			[]byte(string("2222 4000 3000 0004")),
			[]byte(string("5100 0600 0000 0002")),
			[]byte(string("2222 4000 5000 0009")),
			[]byte(string("1354 1001 4004 955")),
			[]byte(string("4111 1111 4555 1142")),
			[]byte(string("4988 4388 4388 4305")),
			[]byte(string("4166 6766 6766 6746")),
			[]byte(string("4646 4646 4646 4644")),
			[]byte(string("4000 6200 0000 0007")),
			[]byte(string("4000 0600 0000 0006")),
			[]byte(string("4293 1891 0000 0008")),
			[]byte(string("4988 0800 0000 0000")),
			[]byte(string("4111 1111 1111 1111")),
			[]byte(string("4444 3333 2222 1111")),
			[]byte(string("4001 5900 0000 0001")),
			[]byte(string("4000 1800 0000 0002")),
			[]byte(string("4000 0200 0000 0000")),
			[]byte(string("4000 1600 0000 0004")),
			[]byte(string("4002 6900 0000 0008")),
			[]byte(string("4400 0000 0000 0008")),
			[]byte(string("4484 6000 0000 0004")),
			[]byte(string("4607 0000 0000 0009")),
			[]byte(string("4977 9494 9494 9497")),
			[]byte(string("4000 6400 0000 0005")),
			[]byte(string("4003 5500 0000 0003")),
			[]byte(string("4000 7600 0000 0001")),
			[]byte(string("4017 3400 0000 0003")),
			[]byte(string("4005 5190 0000 0006")),
			[]byte(string("4131 8400 0000 0003")),
			[]byte(string("4035 5010 0000 0008")),
			[]byte(string("4151 5000 0000 0008")),
			[]byte(string("4571 0000 0000 0001")),
			[]byte(string("4199 3500 0000 0002")),
			[]byte(string("4001 0200 0000 0009")),
		} {
			t.Run("", func(t *testing.T) {
				assert.True(t, IsSensitive(valid), i)
			})
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

func BenchmarkIsSensitive(b *testing.B) {
	run := func(str string, luhn bool) func(b *testing.B) {
		return func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				isSensitive([]byte(str), luhn)
			}
		}
	}

	b.Run("basic", run("4001 0200 0000 0009", false))
	b.Run("luhn", run("4001 0200 0000 0009", true))
}
