package docker

import (
	"strings"
	"time"

	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/go-units"
)

const (
	imagePrefixForV10 = "sha256:"
)

//DurationForHumans returns a human-readable approximation of a duration
//represented as an int64 nanosecond count.
func DurationForHumans(duration int64) string {
	return units.HumanDuration(time.Now().UTC().Sub(
		time.Unix(int64(duration), 0)))

}

//ImageID removes anything that is not part of the ID but is being added
//by the docker library
func ImageID(uglyID string) string {
	id := uglyID
	if strings.HasPrefix(uglyID, imagePrefixForV10) {
		id = strings.TrimPrefix(uglyID, imagePrefixForV10)
	}
	return id
}

//ShortImageID shortens and beutifies an id
func ShortImageID(uglyID string) string {
	return stringid.TruncateID(ImageID(uglyID))
}
