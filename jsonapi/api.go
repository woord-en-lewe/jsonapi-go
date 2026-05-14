package jsonapi

import (
	"encoding/json"
	"errors"
	"fmt"
)

// MediaType is the standard MIME type for JSON:API
const MediaType = "application/vnd.api+json"

// Document represents the top-level JSON:API document.
// Reference: Document Structure -> Top Level
type Document struct {
	JSONAPI *JSONAPIInfo     `json:"jsonapi,omitempty"`
	Data    *json.RawMessage `json:"data,omitempty"` // Can be Object, Array, or Null
	Errors  []ErrorObject    `json:"errors,omitempty"`
	Meta    Meta             `json:"meta,omitempty"`
	Links   *Links           `json:"links,omitempty"`

	// Included represents resources related to the primary data.
	Included []Resource `json:"included,omitempty"`
}

// Resource represents a resource object.
// Reference: Document Structure -> Resource Objects
type Resource struct {
	ID            string                  `json:"id,omitempty"`
	Type          string                  `json:"type"`
	LID           string                  `json:"lid,omitempty"` // Local ID (v1.1)
	Attributes    json.RawMessage         `json:"attributes,omitempty"`
	Relationships map[string]Relationship `json:"relationships,omitempty"`
	Links         *Links                  `json:"links,omitempty"`
	Meta          Meta                    `json:"meta,omitempty"`
}

// ResourceIdentifier represents a resource identifier object (used in linkage).
// Reference: Document Structure -> Resource Identifier Objects
type ResourceIdentifier struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
	LID  string `json:"lid,omitempty"` // Local ID (v1.1)
	Meta Meta   `json:"meta,omitempty"`
}

// Relationship represents a relationship object.
// Reference: Document Structure -> Resource Objects -> Relationships
type Relationship struct {
	Links *Links           `json:"links,omitempty"`
	Data  *json.RawMessage `json:"data,omitempty"` // ResourceIdentifier, []ResourceIdentifier, or Null
	Meta  Meta             `json:"meta,omitempty"`
}

// ErrorObject represents an error object.
// Reference: Errors -> Error Objects
type ErrorObject struct {
	ID     string       `json:"id,omitempty"`
	Links  *Links       `json:"links,omitempty"`
	Status string       `json:"status,omitempty"`
	Code   string       `json:"code,omitempty"`
	Title  string       `json:"title,omitempty"`
	Detail string       `json:"detail,omitempty"`
	Source *ErrorSource `json:"source,omitempty"`
	Meta   Meta         `json:"meta,omitempty"`
}

type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`
	Parameter string `json:"parameter,omitempty"`
	Header    string `json:"header,omitempty"`
}

type JSONAPIInfo struct {
	Version string   `json:"version,omitempty"`
	Ext     []string `json:"ext,omitempty"`
	Profile []string `json:"profile,omitempty"`
	Meta    Meta     `json:"meta,omitempty"`
}

// Meta is a map of non-standard meta-information.
type Meta map[string]any

// Links is a map of link names to Link objects.
type Links struct {
	Self *Link `json:"self,omitempty"`
	Related *Link `json:"related,omitempty"`
	DescribedBy *Link `json:"describedBy,omitempty"`
	First *Link `json:"first,omitempty"`
	Last *Link `json:"last,omitempty"`
	Previous *Link `json:"previous,omitempty"`
	Next *Link `json:"next,omitempty"`
	About *Link `json:"about,omitempty"`
	Type *Link `json:"type,omitempty"`
}

// Link represents a link which can be a string or an object.
// Reference: Document Structure -> Links
type Link struct {
	HRef string
	Rel string
	DescribedBy *Link
	Title string
	Type string
	HRefLang []string
	Meta Meta
}

// Fallback to object
type linkObj struct {
	HRef string `json:"href"`
	Rel string `json:"rel,omitempty"`
	DescribedBy *Link `json:"describedBy,omitempty"`
	Title string `json:"title,omitempty"`
	Type string `json:"type,omitempty"`
	HRefLang json.RawMessage `json:"hreflang,omitempty"`
	Meta Meta   `json:"meta,omitempty"`
}

// UnmarshalJSON handles the fact that a Link can be a string or an object.
func (l *Link) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	// Try unmarshalling as a string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		l.HRef = s
		return nil
	}

	var o linkObj
	if err := json.Unmarshal(data, &o); err != nil {
		return err
	}
	l.HRef = o.HRef
	l.Rel = o.Rel
	l.DescribedBy = o.DescribedBy
	l.Title = o.Title
	l.Type = o.Type

	if string(o.HRefLang) != "null" {
		var err error
		l.HRefLang = func() []string {
			var s string
			if err = json.Unmarshal(o.HRefLang, &s); err == nil {
				return []string{s}
			}

			var ss []string
			if err = json.Unmarshal(o.HRefLang, &ss); err == nil {
				return ss
			}
			return []string{}
		}()
		if err != nil {
			return err
		}
	}

	l.Meta = o.Meta
	return nil
}

// --- Client Helper Methods ---

// HasErrors returns true if the document contains errors.
func (d *Document) HasErrors() bool {
	return len(d.Errors) > 0
}

// UnmarshalDataOne extracts a single resource from the top-level "data".
func (d *Document) UnmarshalDataOne() (*Resource, error) {
	if d.Data == nil {
		return nil, nil
	}
	// Check if it's null (JSON 'null')
	if string(*d.Data) == "null" {
		return nil, nil
	}
	var res Resource
	if err := json.Unmarshal(*d.Data, &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal single resource data: %w", err)
	}
	return &res, nil
}

// UnmarshalDataMany extracts a slice of resources from the top-level "data".
func (d *Document) UnmarshalDataMany() ([]Resource, error) {
	if d.Data == nil {
		return nil, nil
	}
	var res []Resource
	if err := json.Unmarshal(*d.Data, &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource array data: %w", err)
	}
	return res, nil
}

// UnmarshalAttributes unpacks the raw attributes into a provided struct.
func (r *Resource) UnmarshalAttributes(v any) error {
	if r.Attributes == nil {
		return nil
	}
	return json.Unmarshal(r.Attributes, v)
}

// GetIncluded finds a resource in the "included" section matching the given type and ID.
func (d *Document) GetIncluded(resourceType, id string) *Resource {
	for _, inc := range d.Included {
		if inc.Type == resourceType && inc.ID == id {
			return &inc
		}
	}
	return nil
}

// UnmarshalRelationshipOne extracts a single identifier from a relationship.
func (r *Relationship) UnmarshalOne() (*ResourceIdentifier, error) {
	if r.Data == nil {
		return nil, nil
	}
	if string(*r.Data) == "null" {
		return nil, nil
	}
	var ri ResourceIdentifier
	if err := json.Unmarshal(*r.Data, &ri); err != nil {
		return nil, err
	}
	return &ri, nil
}

// UnmarshalRelationshipMany extracts a list of identifiers from a relationship.
func (r *Relationship) UnmarshalMany() ([]ResourceIdentifier, error) {
	if r.Data == nil {
		return nil, nil
	}
	var ris []ResourceIdentifier
	if err := json.Unmarshal(*r.Data, &ris); err != nil {
		return nil, err
	}
	return ris, nil
}

// FindRelated looks up the actual resource for a specific relationship name
// inside the "included" array of the parent document.
func (d *Document) FindRelated(parentResource *Resource, relationName string) (*Resource, error) {
	rel, ok := parentResource.Relationships[relationName]
	if !ok {
		return nil, errors.New("relationship not found")
	}

	identifier, err := rel.UnmarshalOne()
	if err != nil {
		return nil, err
	}
	if identifier == nil {
		return nil, nil // Relationship is empty/null
	}

	included := d.GetIncluded(identifier.Type, identifier.ID)
	if included == nil {
		return nil, fmt.Errorf("resource %s:%s linked but not found in included", identifier.Type, identifier.ID)
	}
	return included, nil
}