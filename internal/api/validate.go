package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// decodeAndValidate decodes JSON body into dst and runs struct validation.
// Returns false and writes 400 error if decode or validation fails.
func decodeAndValidate(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return false
	}
	if err := validate.Struct(dst); err != nil {
		http.Error(w, `{"error":"validation failed"}`, http.StatusBadRequest)
		return false
	}
	return true
}
