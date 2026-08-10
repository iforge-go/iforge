// Package hashid provides site-wide hashids encoding/decoding.
//
// This package exists because the handler-local encodeID/decodeID helpers
// were not reachable from the service package, which needs to compute slugs
// for WebSocket pushes. service cannot import handler (it would create a
// handler -> service -> handler cycle), so the shared encoding lives here.
//
// The hashids instance uses salt="iforge" and minLength=6. NOTE: the salt
// determines encoded slugs; changing it invalidates every existing URL slug.
package hashid

import (
	"errors"

	"github.com/speps/go-hashids"
)

// hashID is the process-level singleton, created once during init.
var hashID *hashids.HashID

func init() {
	hd := hashids.NewData()
	hd.Salt = "iforge"
	hd.MinLength = 6
	h, err := hashids.NewWithData(hd)
	if err != nil {
		panic(err)
	}
	hashID = h
}

// Encode encodes a numeric ID into a URL-safe short string.
// Example: Encode(1) -> "JPl7qK"
func Encode(id int) string {
	result, _ := hashID.Encode([]int{id})
	return result
}

// Decode decodes a hashids string back to the original numeric ID.
// Returns an error for invalid or undecodable input.
func Decode(s string) (int, error) {
	result, err := hashID.DecodeWithError(s)
	if err != nil || len(result) == 0 {
		return 0, errors.New("invalid slug")
	}
	return result[0], nil
}
