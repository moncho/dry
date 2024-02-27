package appui

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/volume"
	units "github.com/docker/go-units"
	"github.com/moncho/dry/docker"
)

const (
	defaultDiskUsageTableFormat = "{{.Type}}\t{{.TotalCount}}\t{{.Active}}\t{{.Size}}\t{{.Reclaimable}}"
)

// DockerDiskUsageRenderer renderer for Docker disk usage
type DockerDiskUsageRenderer struct {
	columns                []string
	diskUsageTableTemplate *template.Template
	diskUsage              *types.DiskUsage
	pruneReport            *docker.PruneReport
	lastPrune              time.Time
	height                 int
	sync.RWMutex
}

// NewDockerDiskUsageRenderer creates a DockerDiskUsageRenderer
func NewDockerDiskUsageRenderer(screenHeight int) *DockerDiskUsageRenderer {
	r := &DockerDiskUsageRenderer{}

	r.columns = []string{
		`TYPE`,
		`TOTAL`,
		`ACTIVE`,
		`SIZE`,
		`RECLAIMABLE`,
	}

	r.diskUsageTableTemplate = buildDiskUsageTableTemplate()
	r.height = screenHeight
	return r
}

// PrepareToRender passes the data to be rendered
func (r *DockerDiskUsageRenderer) PrepareToRender(diskUsage *types.DiskUsage, report *docker.PruneReport) {
	r.Lock()
	r.diskUsage = diskUsage
	if report != nil {
		r.pruneReport = report
		r.lastPrune = time.Now()
	}
	r.Unlock()
}

// Render returns the result of docker system df
func (r *DockerDiskUsageRenderer) String() string {
	r.RLock()
	defer r.RUnlock()
	timeStamp := ""
	if !r.lastPrune.IsZero() {
		timeStamp = r.lastPrune.Format("2006-01-02 15:04:05")
	}
	vars := struct {
		DiskUsageTable string
		Timestamp      string
		PruneTable     string
	}{
		r.diskUsageTable(),
		timeStamp,
		r.pruneTable(),
	}

	var buffer bytes.Buffer
	r.diskUsageTableTemplate.Execute(&buffer, vars)

	return buffer.String()
}

func (r *DockerDiskUsageRenderer) diskUsageTable() string {
	var buffer bytes.Buffer
	t := tabwriter.NewWriter(&buffer, 22, 0, 1, ' ', 0)
	fmt.Fprintln(t, r.tableHeader())
	fmt.Fprint(t, r.formattedDiskUsage())
	t.Flush()
	return buffer.String()
}

func (r *DockerDiskUsageRenderer) pruneTable() string {
	if r.pruneReport == nil {
		return ""
	}
	var buffer bytes.Buffer
	t := tabwriter.NewWriter(&buffer, 22, 0, 1, ' ', 0)

	fmt.Fprintf(t, "Deleted containers: %d \n", len(r.pruneReport.ContainerReport.ContainersDeleted))
	fmt.Fprintf(t, "Deleted images: %d \n", len(r.pruneReport.ImagesReport.ImagesDeleted))
	fmt.Fprintf(t, "Deleted networks: %d \n", len(r.pruneReport.NetworksReport.NetworksDeleted))
	fmt.Fprintf(t, "Deleted volumes: %d \n", len(r.pruneReport.VolumesReport.VolumesDeleted))

	fmt.Fprintf(t, "Total reclaimed space: %s \n", units.HumanSize(float64(r.pruneReport.TotalSpaceReclaimed())))

	t.Flush()
	return buffer.String()
}

func (r *DockerDiskUsageRenderer) tableHeader() string {
	return "<green>" + strings.Join(r.columns, "\t") + "</>"
}

func (r *DockerDiskUsageRenderer) formattedDiskUsage() string {
	du, err := diskUsage(r.diskUsage)
	if err != nil {
		return err.Error()
	}
	return string(du)
}

func buildDiskUsageTableTemplate() *template.Template {
	markup :=
		`{{.DiskUsageTable}}
{{if .Timestamp}}Docker system prune executed on {{.Timestamp}}, results:{{end}}

{{.PruneTable}}
`
	return template.Must(template.New(`diskUsageTable`).Parse(markup))
}

func applyTemplate(tmpl *template.Template, data interface{}) ([]byte, error) {
	var buffer bytes.Buffer

	if err := tmpl.Execute(&buffer, data); err != nil {
		return nil, fmt.Errorf("template execution error: %v", err)
	}

	buffer.WriteString("\n")
	return buffer.Bytes(), nil
}

func diskUsage(diskUsage *types.DiskUsage) ([]byte, error) {
	if diskUsage == nil {
		return nil, errors.New("Disk usage report not provided")
	}
	buffer := bytes.NewBufferString("\n")

	tmpl, err := template.New("").Parse(defaultDiskUsageTableFormat)
	if err != nil {
		return nil, err
	}

	data := []interface{}{
		&diskUsageImagesContext{
			totalSize: diskUsage.LayersSize,
			images:    diskUsage.Images,
		},
		&diskUsageContainersContext{
			containers: diskUsage.Containers,
		},
		&diskUsageVolumesContext{
			volumes: diskUsage.Volumes,
		},
		&diskUsageBuilderContext{
			builderSize: diskUsage.BuilderSize,
		},
	}
	for _, d := range data {
		section, err := applyTemplate(tmpl, d)
		if err != nil {
			return nil, err
		}
		buffer.Write(section)
	}

	return buffer.Bytes(), nil
}

type diskUsageImagesContext struct {
	totalSize int64
	images    []*types.ImageSummary
}

func (c *diskUsageImagesContext) Type() string {
	return "Images"
}

func (c *diskUsageImagesContext) TotalCount() string {
	return fmt.Sprintf("%d", len(c.images))
}

func (c *diskUsageImagesContext) Active() string {
	used := 0
	for _, i := range c.images {
		if i.Containers > 0 {
			used++
		}
	}

	return fmt.Sprintf("%d", used)
}

func (c *diskUsageImagesContext) Size() string {
	return units.HumanSize(float64(c.totalSize))

}

func (c *diskUsageImagesContext) Reclaimable() string {
	var used int64

	for _, i := range c.images {
		if i.Containers != 0 {
			if i.VirtualSize == -1 || i.SharedSize == -1 {
				continue
			}
			used += i.VirtualSize - i.SharedSize
		}
	}

	reclaimable := c.totalSize - used
	if c.totalSize > 0 {
		return fmt.Sprintf("%s (%v%%)", units.HumanSize(float64(reclaimable)), (reclaimable*100)/c.totalSize)
	}
	return units.HumanSize(float64(reclaimable))
}

type diskUsageContainersContext struct {
	containers []*types.Container
}

func (c *diskUsageContainersContext) Type() string {
	return "Containers"
}

func (c *diskUsageContainersContext) TotalCount() string {
	return fmt.Sprintf("%d", len(c.containers))
}

func (c *diskUsageContainersContext) isActive(container types.Container) bool {
	return strings.Contains(container.State, "running") ||
		strings.Contains(container.State, "paused") ||
		strings.Contains(container.State, "restarting")
}

func (c *diskUsageContainersContext) Active() string {
	used := 0
	for _, container := range c.containers {
		if c.isActive(*container) {
			used++
		}
	}

	return fmt.Sprintf("%d", used)
}

func (c *diskUsageContainersContext) Size() string {
	var size int64

	for _, container := range c.containers {
		size += container.SizeRw
	}

	return units.HumanSize(float64(size))
}

func (c *diskUsageContainersContext) Reclaimable() string {
	var reclaimable int64
	var totalSize int64

	for _, container := range c.containers {
		if !c.isActive(*container) {
			reclaimable += container.SizeRw
		}
		totalSize += container.SizeRw
	}

	if totalSize > 0 {
		return fmt.Sprintf("%s (%v%%)", units.HumanSize(float64(reclaimable)), (reclaimable*100)/totalSize)
	}

	return units.HumanSize(float64(reclaimable))
}

type diskUsageVolumesContext struct {
	volumes []*volume.Volume
}

func (c *diskUsageVolumesContext) Type() string {
	return "Local Volumes"
}

func (c *diskUsageVolumesContext) TotalCount() string {
	return fmt.Sprintf("%d", len(c.volumes))
}

func (c *diskUsageVolumesContext) Active() string {

	used := 0
	for _, v := range c.volumes {
		if v.UsageData.RefCount > 0 {
			used++
		}
	}

	return fmt.Sprintf("%d", used)
}

func (c *diskUsageVolumesContext) Size() string {
	var size int64

	for _, v := range c.volumes {
		if v.UsageData.Size != -1 {
			size += v.UsageData.Size
		}
	}

	return units.HumanSize(float64(size))
}

func (c *diskUsageVolumesContext) Reclaimable() string {
	var reclaimable int64
	var totalSize int64

	for _, v := range c.volumes {
		if v.UsageData.Size != -1 {
			if v.UsageData.RefCount == 0 {
				reclaimable += v.UsageData.Size
			}
			totalSize += v.UsageData.Size
		}
	}

	if totalSize > 0 {
		return fmt.Sprintf("%s (%v%%)", units.HumanSize(float64(reclaimable)), (reclaimable*100)/totalSize)
	}

	return units.HumanSize(float64(reclaimable))
}

type diskUsageBuilderContext struct {
	builderSize int64
}

func (c *diskUsageBuilderContext) Type() string {
	return "Build Cache"
}

func (c *diskUsageBuilderContext) TotalCount() string {
	return ""
}

func (c *diskUsageBuilderContext) Active() string {
	return ""
}

func (c *diskUsageBuilderContext) Size() string {
	return units.HumanSize(float64(c.builderSize))
}

func (c *diskUsageBuilderContext) Reclaimable() string {
	return c.Size()
}
