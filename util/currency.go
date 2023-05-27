package util

var ValidCurrencies []string = []string{
	"USD",
	"CAD",
	"EUR",
}

func ValidCurrency(currency string) bool {
	for _, cur := range ValidCurrencies {
		if cur == currency {
			return true
		}
	}

	return false
}
