package api

import (
	"github.com/aryan-more/simple_bank/util"
	"github.com/go-playground/validator/v10"
)

var validCurrency validator.Func = func(fl validator.FieldLevel) bool {
	currency, ok := fl.Field().Interface().(string)
	if ok && util.ValidCurrency(currency) {
		return true
	}

	return false
}
