package httpx

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = func() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return field.Name
		}
		return name
	})
	return v
}()

func Validate(v any) error {
	err := validate.Struct(v)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return Internal(fmt.Errorf("validasi: %w", err))
	}
	fields := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		if _, exists := fields[fe.Field()]; exists {
			continue
		}
		fields[fe.Field()] = messageFor(fe)
	}
	return Validation(fields)
}

func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "Wajib diisi."
	case "min":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("Minimal %s karakter.", fe.Param())
		}
		return fmt.Sprintf("Minimal %s.", fe.Param())
	case "max":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("Maksimal %s karakter.", fe.Param())
		}
		return fmt.Sprintf("Maksimal %s.", fe.Param())
	case "gte":
		return fmt.Sprintf("Minimal %s.", fe.Param())
	case "lte":
		return fmt.Sprintf("Maksimal %s.", fe.Param())
	case "oneof":
		return "Pilihan tidak tersedia."
	case "uuid", "uuid4":
		return "Format ID tidak valid."
	case "datetime":
		return "Format tanggal tidak valid."
	case "e164", "numeric":
		return "Nomor tidak valid."
	case "latitude", "longitude":
		return "Koordinat tidak valid."
	}
	return "Nilai tidak valid."
}
