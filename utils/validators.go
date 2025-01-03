package utils

import (
	"regexp"
	"slices"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// YYYY-MM-dd
var DayDateFmt = "2006-01-02"

var ValidTaskStatuses = []string{
	"Won't do",
	"To do",
	"In progress",
	"Done",
}

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
	return false
}

var taskStatusValidator validator.Func = func(fl validator.FieldLevel) bool {
	status, ok := fl.Field().Interface().(string)
	if ok {
		return slices.Contains(ValidTaskStatuses, status)
	}
	return false
}

var dayDateFormatValidator validator.Func = func(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(string)
	if ok {
		_, err := time.Parse(DayDateFmt, date)
		return err == nil
	}
	return false
}

func RegisterValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("strongpass", strongPasswordValidator)
		v.RegisterValidation("taskStatus", taskStatusValidator)
		v.RegisterValidation("dayFormat", dayDateFormatValidator)
	}
}
