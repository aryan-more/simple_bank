package util

var validCurrency []string = []string{
	"USD",
	"CAD",
	"EUR",
}

func ValidCurrency(currency string) bool {
	for _, cur := range validCurrency {
		if cur == currency {
			return true
		}
	}

	return false
}
