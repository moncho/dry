package docker

import (
	"strings"
	"time"

	"github.com/docker/go-units"
)

const (
	imagePrefixForV10 = "sha256:"
	//ShortLen defines the default size of shortened ID
	ShortLen = 12
)

// DurationForHumans returns a human-readable approximation of a duration
// represented as an int64 nanosecond count.
func DurationForHumans(duration int64) string {
	return units.HumanDuration(time.Now().UTC().Sub(
		time.Unix(duration, 0)))

}

// ImageID removes anything that is not part of the ID but is being added
// by the docker library
func ImageID(uglyID string) string {
	id := uglyID
	if strings.HasPrefix(uglyID, imagePrefixForV10) {
		id = strings.TrimPrefix(uglyID, imagePrefixForV10)
	}
	return id
}

// ShortImageID shortens and beutifies an id
func ShortImageID(uglyID string) string {
	return TruncateID(ImageID(uglyID))
}

// TruncateID returns a shorthand version of a string identifier for convenience.
// A collision with other shorthands is very unlikely, but possible.
// In case of a collision a lookup with TruncIndex.Get() will fail, and the caller
// will need to use a longer prefix, or the full-length Id.
func TruncateID(id string) string {
	if i := strings.IndexRune(id, ':'); i >= 0 {
		id = id[i+1:]
	}
	trimTo := ShortLen
	if len(id) < ShortLen {
		trimTo = len(id)
	}
	return id[:trimTo]
}
