package edge

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// SendJSONResponse sends a JSON response with the given body
func SendJSONResponse(res http.ResponseWriter, body interface{}) {
	res.Header().Set("Content-type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(body)
}

// SendErrorResponse sends a JSON error response with the given error
func SendErrorResponse(res http.ResponseWriter, err error) {
	// log the error message
	log.Println(err)

	apiError, ok := err.(Error)
	status := http.StatusInternalServerError
	if ok {
		status = apiError.GetStatus()
	} else {
		apiError = NewInternalError("Internal Server Error")
	}

	res.Header().Set("Content-type", "application/json")
	res.WriteHeader(status)
	json.NewEncoder(res).Encode(apiError)
}

func ParseJSONBody(body io.Reader, obj interface{}) error {
	reflectVal := reflect.ValueOf(obj)
	if reflectVal.Kind() != reflect.Ptr {
		log.Println("Second argument to ParseJSONBody must be a reference")
		return NewInternalError("Internal server error")
	}

	err := json.NewDecoder(body).Decode(&obj)
	if err != nil {
		switch err.(type) {
		case *json.UnmarshalTypeError:
			jsonError := err.(*json.UnmarshalTypeError)
			return NewInvalidParameterError(jsonError.Field, fmt.Sprintf("must be %s", primitiveTypeToDisplayName(jsonError.Type)))
		default:
			return NewInvalidRequestError("Invalid request body")
		}
	}

	err = validate.Struct(obj)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			objType := reflect.ValueOf(obj).Elem().Type()
			invalidField, fieldFound := objType.FieldByName(err.Field())
			if !fieldFound {
				return NewInvalidRequestError("Invalid request body")
			}

			fieldName := invalidField.Tag.Get("json")
			validationRules := make(map[string]string)
			validationRulesParts := strings.Split(invalidField.Tag.Get("validate"), ",")
			for _, validationRulesPart := range validationRulesParts {
				ruleParts := strings.Split(validationRulesPart, "=")
				if len(ruleParts) > 1 {
					validationRules[ruleParts[0]] = ruleParts[1]
				}
			}

			ruleName := err.Tag()
			switch ruleName {
			case "max":
				return NewInvalidParameterError(fieldName, fmt.Sprintf("must be less than %s", validationRules[ruleName]))
			case "min":
				return NewInvalidParameterError(fieldName, fmt.Sprintf("must be greater than %s", validationRules[ruleName]))
			case "required":
				return NewMissingRequiredParameterError(fieldName)
			default:
				return NewInvalidRequestError("Invalid request body")
			}
		}
	}

	return nil
}

func primitiveTypeToDisplayName(primitiveType reflect.Type) string {
	switch fmt.Sprint(primitiveType) {
	case "bool":
		return "true or false"
	case "string":
		return "a string"
	case "int":
		return "a number"
	case "int8":
		return "a number"
	case "int16":
		return "a number"
	case "int32":
		return "a number"
	case "int64":
		return "a number"
	case "uint":
		return "a number"
	case "uint8":
		return "a number"
	case "uint16":
		return "a number"
	case "uint32":
		return "a number"
	case "uint64":
		return "a number"
	case "uintptr":
		return "a number"
	case "float32":
		return "a decimal"
	case "float64":
		return "a decimal"
	default:
		return fmt.Sprintf("type %s", primitiveType)
	}
}
