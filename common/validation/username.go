package validation

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var usernameRegexp = regexp.MustCompile("^[a-zA-Z0-9]+([._-][a-zA-Z0-9]+)*$")

var IsValidUsername validator.Func = func(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if len(value) < 4 || len(value) > 20 {
		return false
	}
	return usernameRegexp.MatchString(value)
}
