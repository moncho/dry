package formatter

import (
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/go-units"
	"github.com/moncho/dry/docker"
)

const (
	imageIDHeader = "IMAGE ID"
	repository    = "REPOSITORY"
	tag           = "TAG"
	digest        = "DIGEST"
	createdSince  = "CREATEDSINCE"
	size          = "SIZE"
)

// ImageFormatter knows how to pretty-print the information of an image
type ImageFormatter struct {
	trunc  bool
	header []string
	image  types.ImageSummary
}

// NewImageFormatter creates an image formatter
func NewImageFormatter(image types.ImageSummary, trunc bool) *ImageFormatter {
	return &ImageFormatter{trunc: trunc, image: image}
}

func (formatter *ImageFormatter) addHeader(header string) {
	if formatter.header == nil {
		formatter.header = []string{}
	}
	formatter.header = append(formatter.header, strings.ToUpper(header))
}

// ID prettifies the id
func (formatter *ImageFormatter) ID() string {
	formatter.addHeader(imageIDHeader)
	if formatter.trunc {
		return docker.TruncateID(docker.ImageID(formatter.image.ID))
	}
	return docker.ImageID(formatter.image.ID)
}

// Repository prettifies the repository
func (formatter *ImageFormatter) Repository() string {
	formatter.addHeader(repository)
	if len(formatter.image.RepoTags) > 0 {
		tagPos := strings.LastIndex(formatter.image.RepoTags[0], ":")
		if tagPos > 0 {
			return formatter.image.RepoTags[0][:tagPos]
		}
		return formatter.image.RepoTags[0]
	} else if len(formatter.image.RepoDigests) > 0 {
		tagPos := strings.LastIndex(formatter.image.RepoDigests[0], "@")
		if tagPos > 0 {
			return formatter.image.RepoDigests[0][:tagPos]
		}
		return formatter.image.RepoDigests[0]
	}

	return "<none>"
}

// Tag prettifies the tag
func (formatter *ImageFormatter) Tag() string {
	formatter.addHeader(tag)
	if len(formatter.image.RepoTags) > 0 {
		tagPos := strings.LastIndex(formatter.image.RepoTags[0], ":")
		return formatter.image.RepoTags[0][tagPos+1:]
	}
	return "<none>"

}

// Digest prettifies the image digestv
func (formatter *ImageFormatter) Digest() string {
	formatter.addHeader(digest)
	if len(formatter.image.RepoDigests) == 0 {
		return ""
	}
	return formatter.image.RepoDigests[0]
}

// CreatedSince prettifies the image creation date
func (formatter *ImageFormatter) CreatedSince() string {
	formatter.addHeader(createdSince)

	return docker.DurationForHumans(formatter.image.Created)
}

// Size prettifies the image size
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
