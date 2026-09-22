package core_http_request

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	core_error "github.com/Aam-Shaegar/Rhythm/internal/core/errors"

	"github.com/go-playground/validator/v10"
)

var requestValidator = validator.New()

type validatable interface {
	Validate() error
}

func DecodeAndValidateRequest(r *http.Request, dest any) error {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		return fmt.Errorf("decode json: %v: %w", err, core_error.ErrInvalidArgument)
	}

	v, ok := dest.(validatable)
	if ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("request validation: %v: %w", err, core_error.ErrInvalidArgument)
		}
	} else {
		if err := requestValidator.Struct(dest); err != nil {
			return fmt.Errorf("request validation: %v: %w", err, core_error.ErrInvalidArgument)
		}
	}

	return nil
}

func DecodeQueryParams(r *http.Request, dest any) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to struct: %w", core_error.ErrInvalidArgument)
	}

	t := v.Elem().Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("query")
		if tag == "" || tag == "-" {
			continue
		}

		value := r.URL.Query().Get(tag)
		if value == "" {
			continue
		}

		fieldValue := v.Elem().Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		if err := setFieldFromString(fieldValue, value); err != nil {
			return fmt.Errorf("query param '%s': %v: %w", tag, err, core_error.ErrInvalidArgument)
		}
	}

	if err := requestValidator.Struct(dest); err != nil {
		return fmt.Errorf("request validation: %v: %w", err, core_error.ErrInvalidArgument)
	}

	return nil
}

func setFieldFromString(field reflect.Value, value string) error {
	if field.Type() == reflect.TypeOf(time.Time{}) {
		for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02T15:04:05Z07:00"} {
			if t, err := time.Parse(layout, value); err == nil {
				field.Set(reflect.ValueOf(t))
				return nil
			}
		}
		return fmt.Errorf("not a valid time: %q", value)
	}
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intVal int64
		var err error
		if field.Type().String() == "time.Duration" {
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(d))
			return nil
		}
		intVal, err = strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldFromString(field.Elem(), value)
	default:
		return fmt.Errorf("unsupported type %s", field.Kind())
	}
	return nil
}
