package jasco

import (
	"fmt"
	"github.com/gocraft/web"
	"strconv"
)

// PathParams contains parameters embedded in the URL supported by gocraft/web
type PathParams struct {
	req *web.Request
}

// String returns a string value corresponding to key. When the key doesn't
// exist, it returns defaultValue.
func (p *PathParams) String(key string, defaultValue string) string {
	s, ok := p.req.PathParams[key]
	if !ok {
		return defaultValue
	}
	return s
}

// RequiredString return a string value corresponding to the key. When the key
// doesn't exist, it returns an error.
func (p *PathParams) RequiredString(key string) (string, error) {
	s, ok := p.req.PathParams[key]
	if !ok {
		return "", fmt.Errorf("path parameter '%v' doesn't exist", key)
	}
	return s, nil
}

// Int returns a positive integer value correspoding to the key as uint64. When
// the key doesn't exist, it returns defaultValue. It returns an error if it
// cannot parse the path string parameter as an integer or the value is greater
// than or equal to 2^63.
func (p *PathParams) Int(key string, defaultValue uint64) (uint64, error) {
	s, err := p.RequiredString(key)
	if err != nil {
		return defaultValue, nil
	}
	return strconv.ParseUint(s, 10, 63)
}

func (p *PathParams) RequiredInt(key string) (uint64, error) {
	s, err := p.RequiredString(key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(s, 10, 63)
}
