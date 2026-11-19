package dna

import (
	"bytes"
	"encoding/json"
	"fmt"
	"iter"
	"strings"
	"time"
)

// Time is a timestamp sent by the API.
//
// The API sends ISO 8601 times without an offset ("2024-03-15T10:20:30"),
// which time.Time cannot decode. Time accepts those (as UTC) and RFC 3339.
// Missing, null, "0001-01-01T00:00:00" and unparseable values decode to the
// zero time instead of failing the whole response: check IsZero.
type Time struct{ time.Time }

var timeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02",
}

func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Time) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) != nil {
		t.Time = time.Time{} // null, numbers, ...
		return nil
	}
	t.Time = parseTime(s)
	return nil
}

// MarshalJSON implements json.Marshaler: RFC 3339, or null for the zero time.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.UTC().Format(time.RFC3339Nano))
}

// String formats the time as RFC 3339, or "" for the zero time.
func (t Time) String() string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// Attributes are registry-specific key/value pairs, such as the extra
// attributes some TLDs need for a registration.
//
// When decoding, Attributes accepts an object (values of any type are
// turned into strings), and also the empty array or null the API sometimes
// sends instead of an empty object.
type Attributes map[string]string

// UnmarshalJSON implements json.Unmarshaler.
func (a *Attributes) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || b[0] != '{' {
		*a = nil // null, [] or something unexpected
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	out := make(Attributes, len(raw))
	for k, v := range raw {
		switch x := v.(type) {
		case nil:
			out[k] = ""
		case string:
			out[k] = x
		default:
			enc, _ := json.Marshal(x)
			out[k] = strings.Trim(string(enc), `"`)
		}
	}
	*a = out
	return nil
}

// Page is one page of a list endpoint.
type Page[T any] struct {
	// Items are the items of this page.
	Items []T `json:"items"`
	// TotalCount is the number of items across all pages.
	TotalCount int64 `json:"totalCount"`
}

// defaultPageSize is the page size the iterators ask for.
const defaultPageSize = 100

// paginate returns an iterator over every item, fetching pages with fetch.
// It stops at the first error, after yielding it.
func paginate[T any](opts ListOptions, fetch func(ListOptions) (*Page[T], error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		if opts.Limit <= 0 {
			opts.Limit = defaultPageSize
		}
		for {
			page, err := fetch(opts)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			for _, item := range page.Items {
				if !yield(item, nil) {
					return
				}
			}
			next := opts.Skip + len(page.Items)
			switch {
			case len(page.Items) == 0:
				return
			case page.TotalCount > 0 && int64(next) >= page.TotalCount:
				return
			case page.TotalCount <= 0 && len(page.Items) < opts.Limit:
				return
			}
			opts.Skip = next
		}
	}
}

// collect drains an iterator into a slice.
func collect[T any](seq iter.Seq2[T, error]) ([]T, error) {
	var out []T
	for item, err := range seq {
		if err != nil {
			return out, err
		}
		out = append(out, item)
	}
	return out, nil
}

// Price is the price of a product for one period.
type Price struct {
	// Period is the number of years.
	Period int `json:"period"`
	// Price is the amount.
	Price float64 `json:"price"`
	// Currency is the ISO 4217 code, e.g. "USD".
	Currency string `json:"currency"`
}

// String formats the price as "11.99 USD".
func (p Price) String() string {
	return strings.TrimSpace(fmt.Sprintf("%.2f %s", p.Price, p.Currency))
}

func boolPtr(b bool) *bool { return &b }

// Bool returns a pointer to b, for optional fields such as
// Contact.DiscloseFlag.
func Bool(b bool) *bool { return boolPtr(b) }
