package services

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/ismaelucky94/near_u_exercise/internal/repositories"
)

// encodeCursor serialises a pagination cursor to a URL-safe base64 string.
func encodeCursor(c repositories.ListCursor) string {
	b, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(b)
}

// decodeCursor deserialises a pagination cursor.
func decodeCursor(s string) (repositories.ListCursor, error) {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return repositories.ListCursor{}, err
	}
	var c repositories.ListCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return repositories.ListCursor{}, err
	}
	return c, nil
}

// dateOnly returns a UTC timestamp at midnight for the given date.
func dateOnly(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
