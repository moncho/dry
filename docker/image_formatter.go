package docker

import (
	"strings"

	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/go-units"
	"github.com/fsouza/go-dockerclient"
)

const (
	imageIDHeader = "IMAGE ID"
	repository    = "REPOSITORY"
	tag           = "TAG"
	digest        = "DIGEST"
	createdSince  = "CREATEDSINCE"
	size          = "SIZE"
)

//ImageFormatter knows how to pretty-print the information of an image
type ImageFormatter struct {
	trunc  bool
	header []string
	image  docker.APIImages
}

func (formatter *ImageFormatter) addHeader(header string) {
	if formatter.header == nil {
		formatter.header = []string{}
	}
	formatter.header = append(formatter.header, strings.ToUpper(header))
}

//ID prettifies the id
func (formatter *ImageFormatter) ID() string {
	formatter.addHeader(imageIDHeader)
	if formatter.trunc {
		return stringid.TruncateID(ImageID(formatter.image.ID))
	}
	return ImageID(formatter.image.ID)
}

//Repository prettifies the repository
func (formatter *ImageFormatter) Repository() string {
	formatter.addHeader(repository)
	if len(formatter.image.RepoTags) > 0 {
		return strings.Split(formatter.image.RepoTags[0], ":")[0]
	}
	return ""
}

//Tag prettifies the tag
func (formatter *ImageFormatter) Tag() string {
	formatter.addHeader(tag)
	if len(formatter.image.RepoTags) > 0 {
		return strings.Split(formatter.image.RepoTags[0], ":")[1]
	}
	return ""

}

//Digest prettifies the image digestv
func (formatter *ImageFormatter) Digest() string {
	formatter.addHeader(digest)
	if len(formatter.image.RepoDigests) == 0 {
		return ""
	}
	return formatter.image.RepoDigests[0]
}

//CreatedSince prettifies the image creation date
func (formatter *ImageFormatter) CreatedSince() string {
	formatter.addHeader(createdSince)

	return DurationForHumans(int64(formatter.image.Created))
}

//Size prettifies the image size
func (formatter *ImageFormatter) Size() string {

	formatter.addHeader(size)
	//srw := units.HumanSize(float64(formatter.image.Size))
	//sf := srw

	if formatter.image.VirtualSize > 0 {
		sv := units.HumanSize(float64(formatter.image.VirtualSize))
		//sf = fmt.Sprintf("%s (virtual %s)", srw, sv)
		return sv
	}
	return ""
}
