package model

import (
	"encoding/json"
	"time"
)

// Variant describes the resource schema selected by VARIANT.
type Variant struct {
	Name           string
	Resource       string
	RequiredFields []string
	OptionalInt    string
	OptionalText   string
}

// FilterField returns the fourth required field, which is also the list filter.
func (v Variant) FilterField() string {
	return v.RequiredFields[3]
}

// Resource is the typed domain representation of one stored item.
// Fields contains the variant-specific JSON fields and is flattened during encoding.
type Resource struct {
	ID        uint32         `json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Fields    map[string]any `json:"-"`
}

// MarshalJSON flattens the domain resource into the API JSON representation.
func (r Resource) MarshalJSON() ([]byte, error) {
	fields := make(map[string]any, len(r.Fields)+3)
	for key, value := range r.Fields {
		fields[key] = value
	}
	fields["id"] = r.ID
	fields["created_at"] = r.CreatedAt
	fields["updated_at"] = r.UpdatedAt
	return json.Marshal(fields)
}

// Clone returns a copy of the resource and its variant-specific fields.
func (r Resource) Clone() Resource {
	r.Fields = cloneMap(r.Fields)
	return r
}

func cloneMap(fields map[string]any) map[string]any {
	clone := make(map[string]any, len(fields))
	for key, value := range fields {
		clone[key] = value
	}
	return clone
}
