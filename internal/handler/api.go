package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"homework/internal/model"
)

const maxRequestBodyBytes = 1 << 20

// Store is the repository contract used by the HTTP API.
type Store interface {
	Create(fields map[string]any) (model.Resource, error)
	Get(id uint32) (model.Resource, bool)
	Update(id uint32, fields map[string]any) (model.Resource, bool)
	Delete(id uint32) bool
	List(filter string) []model.Resource
}

// API handles all routes for one resource variant.
type API struct {
	store         Store
	resourceName  string
	requiredField []string
	optionalInt   string
	optionalText  string
}

// New builds a resource handler.
func New(store Store, resourceName string, requiredFields []string, optionalInt, optionalText string) *API {
	return &API{
		store:         store,
		resourceName:  resourceName,
		requiredField: requiredFields,
		optionalInt:   optionalInt,
		optionalText:  optionalText,
	}
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/"+a.resourceName)
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			a.list(w, r)
		case http.MethodPost:
			a.create(w, r)
		case http.MethodOptions:
			writePreflight(w)
		default:
			a.error(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
		return
	}

	if !strings.HasPrefix(path, "/") {
		a.error(w, http.StatusNotFound, "not_found", a.resourceName+" not found")
		return
	}
	if r.Method == http.MethodOptions {
		writePreflight(w)
		return
	}
	resourceID := strings.TrimPrefix(path, "/")
	if strings.Contains(resourceID, "/") {
		a.error(w, http.StatusNotFound, "not_found", a.resourceName+" not found")
		return
	}
	id, err := parseID(resourceID)
	if err != nil {
		a.error(w, http.StatusBadRequest, "invalid_id", "ID must be an unsigned integer")
		return
	}

	switch r.Method {
	case http.MethodGet:
		a.get(w, id)
	case http.MethodPut:
		a.update(w, r, id)
	case http.MethodDelete:
		a.delete(w, id)
	default:
		a.error(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	}
}

func (a *API) create(w http.ResponseWriter, r *http.Request) {
	fields, err := decodeValidBody(w, r, a.requiredField, a.optionalInt, a.optionalText)
	if err != nil {
		a.error(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	item, err := a.store.Create(fields)
	if err != nil {
		a.error(w, http.StatusInternalServerError, "internal_server_error", "Internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) get(w http.ResponseWriter, id uint32) {
	item, ok := a.store.Get(id)
	if !ok {
		a.error(w, http.StatusNotFound, "not_found", a.resourceName+" not found")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) update(w http.ResponseWriter, r *http.Request, id uint32) {
	fields, err := decodeValidBody(w, r, a.requiredField, a.optionalInt, a.optionalText)
	if err != nil {
		a.error(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	item, ok := a.store.Update(id, fields)
	if !ok {
		a.error(w, http.StatusNotFound, "not_found", a.resourceName+" not found")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) delete(w http.ResponseWriter, id uint32) {
	if !a.store.Delete(id) {
		a.error(w, http.StatusNotFound, "not_found", a.resourceName+" not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) list(w http.ResponseWriter, r *http.Request) {
	page, limit, ok := parsePagination(r.URL.Query())
	if !ok {
		a.error(w, http.StatusBadRequest, "invalid_pagination", "page and limit must be positive integers")
		return
	}
	items := a.store.List(r.URL.Query().Get(a.requiredField[3]))
	if page > ^uint64(0)/limit {
		items = []model.Resource{}
	} else {
		start := (page - 1) * limit
		if start >= uint64(len(items)) {
			items = []model.Resource{}
		} else {
			end := start + limit
			if end > uint64(len(items)) {
				end = uint64(len(items))
			}
			items = items[start:end]
		}
	}
	writeJSON(w, http.StatusOK, items)
}

func decodeValidBody(w http.ResponseWriter, r *http.Request, required []string, optionalInt, optionalText string) (map[string]any, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	var body any
	if err := decoder.Decode(&body); err != nil {
		return nil, fmt.Errorf("request body must be a valid JSON object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("request body must be a valid JSON object")
	}
	object, ok := body.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("request body must be a JSON object")
	}

	fields := make(map[string]any, len(required)+2)
	for _, field := range required {
		value, present := object[field]
		if !present || !isNonEmptyString(value) {
			return nil, fmt.Errorf("field %s is required and must be a non-empty string", field)
		}
		fields[field] = value
	}
	fields[optionalInt] = nil
	fields[optionalText] = nil
	if value, present := object[optionalInt]; present {
		if value == nil {
			fields[optionalInt] = nil
		} else if !isIntegerNumber(value) {
			return nil, fmt.Errorf("field %s must be an integer", optionalInt)
		} else {
			fields[optionalInt] = value
		}
	}
	if value, present := object[optionalText]; present {
		if value == nil {
			fields[optionalText] = nil
		} else if !isString(value) {
			return nil, fmt.Errorf("field %s must be a string", optionalText)
		} else {
			fields[optionalText] = value
		}
	}
	return fields, nil
}

func isNonEmptyString(value any) bool {
	text, ok := value.(string)
	return ok && strings.TrimSpace(text) != ""
}

func isString(value any) bool {
	_, ok := value.(string)
	return ok
}

func isIntegerNumber(value any) bool {
	number, ok := value.(json.Number)
	if !ok {
		return false
	}
	_, err := number.Int64()
	return err == nil
}

func parseID(value string) (uint32, error) {
	if value == "" || strings.ContainsAny(value, "+-") {
		return 0, fmt.Errorf("invalid id")
	}
	id, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

func parsePagination(query map[string][]string) (uint64, uint64, bool) {
	page := uint64(1)
	limit := uint64(10)
	if values := query["page"]; len(values) > 0 {
		if values[0] == "" {
			return 0, 0, false
		}
		parsed, err := strconv.ParseUint(values[0], 10, 64)
		if err != nil || parsed < 1 {
			return 0, 0, false
		}
		page = parsed
	}
	if values := query["limit"]; len(values) > 0 {
		if values[0] == "" {
			return 0, 0, false
		}
		parsed, err := strconv.ParseUint(values[0], 10, 64)
		if err != nil || parsed < 1 {
			return 0, 0, false
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}
	return page, limit, true
}

func (a *API) error(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writePreflight(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Origin")
	w.WriteHeader(http.StatusNoContent)
}
