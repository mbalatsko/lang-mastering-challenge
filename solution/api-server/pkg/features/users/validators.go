package users

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var strongPasswordValidator validator.Func = func(fl validator.FieldLevel) bool {
	password, ok := fl.Field().Interface().(string)
	if ok {
		var (
			lengthValid      = len(password) >= 8 && len(password) <= 20 // 8-20 characters
			lowercaseRegex   = regexp.MustCompile(`[a-z]`)               // At least one lowercase
			uppercaseRegex   = regexp.MustCompile(`[A-Z]`)               // At least one uppercase
			digitRegex       = regexp.MustCompile(`\d`)                  // At least one digit
			specialCharRegex = regexp.MustCompile(`[!@#$%^&*()-_+=<>?]`) // At least one special character
		)

		return lengthValid &&
			lowercaseRegex.MatchString(password) &&
			uppercaseRegex.MatchString(password) &&
			digitRegex.MatchString(password) &&
			specialCharRegex.MatchString(password)
	}
	return true
}

func RegisterUsersValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("strongpass", strongPasswordValidator)
	}
}
