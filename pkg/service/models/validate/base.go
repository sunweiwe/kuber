package validate

import (
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

var instance *Validator

func Get() *Validator {
	return instance
}

type Validator struct {
	Validator  *validator.Validate
	Translator ut.Translator
	db         *gorm.DB
}
