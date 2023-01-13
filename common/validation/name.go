package validation

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var nameRegexp = regexp.MustCompile("^[a-zA-Z0-9]+([._-][a-zA-Z0-9]+)*$")

// TODO implement translation
var IsValidName validator.Func = func(fl validator.FieldLevel) bool {
	return nameRegexp.MatchString(fl.Field().String())
}
