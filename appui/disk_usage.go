package appui

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/tabwriter"
	"text/template"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types"
	units "github.com/docker/go-units"
	"github.com/moncho/dry/docker"
)

type diskUsageColumn struct {
	name  string // The name of the field in the struct.
	title string // Title to display in the tableHeader.
}

//DockerDiskUsageRenderer renderer for Docker disk usage
type DockerDiskUsageRenderer struct {
	columns                []diskUsageColumn
	diskUsageTableTemplate *template.Template
	diskUsage              *types.DiskUsage
	pruneReport            *docker.PruneReport
	height                 int
	sync.RWMutex
}

//NewDockerDiskUsageRenderer creates a DockerDiskUsageRenderer
func NewDockerDiskUsageRenderer(screenHeight int) *DockerDiskUsageRenderer {
	r := &DockerDiskUsageRenderer{}

	r.columns = []diskUsageColumn{
		{`Type`, `TYPE`},
		{`Total`, `TOTAL`},
		{`Active`, `ACTIVE`},
		{`Size`, `SIZE`},
		{`Reclaimable`, `RECLAIMABLE`},
	}

	r.diskUsageTableTemplate = buildDiskUsageTableTemplate()
	r.height = screenHeight
	return r
}

//PrepareToRender passes the data to be rendered
func (r *DockerDiskUsageRenderer) PrepareToRender(diskUsage *types.DiskUsage, report *docker.PruneReport) {
	r.Lock()
	r.diskUsage = diskUsage
	r.pruneReport = report
	r.Unlock()
}

//Render returns the result of docker system df
func (r *DockerDiskUsageRenderer) Render() string {
	r.RLock()
	defer r.RUnlock()
	vars := struct {
		DiskUsageTable string
		PruneTable     string
	}{
		r.diskUsageTable(),
		r.pruneTable(),
	}

	buffer := new(bytes.Buffer)
	r.diskUsageTableTemplate.Execute(buffer, vars)

	return buffer.String()
}

func (r *DockerDiskUsageRenderer) diskUsageTable() string {
	buffer := new(bytes.Buffer)
	t := tabwriter.NewWriter(buffer, 22, 0, 1, ' ', 0)
	replacer := strings.NewReplacer(`\t`, "\t", `\n`, "\n")
	fmt.Fprintln(t, replacer.Replace(r.tableHeader()))
	fmt.Fprint(t, replacer.Replace(r.formattedDiskUsage()))
	t.Flush()
	return buffer.String()
}

func (r *DockerDiskUsageRenderer) pruneTable() string {
	if r.pruneReport == nil {
		return ""
	}
	buffer := new(bytes.Buffer)
	t := tabwriter.NewWriter(buffer, 22, 0, 1, ' ', 0)

	fmt.Fprintf(t, "Deleted containers: %d \n", len(r.pruneReport.ContainerReport.ContainersDeleted))
	fmt.Fprintf(t, "Deleted images: %d \n", len(r.pruneReport.ImagesReport.ImagesDeleted))
	fmt.Fprintf(t, "Deleted networks: %d \n", len(r.pruneReport.NetworksReport.NetworksDeleted))
	fmt.Fprintf(t, "Deleted volumes: %d \n", len(r.pruneReport.VolumesReport.VolumesDeleted))

	fmt.Fprintf(t, "Total reclaimed space: %s \n", units.HumanSize(float64(r.pruneReport.TotalSpaceReclaimed())))

	t.Flush()
	return buffer.String()
}

func (r *DockerDiskUsageRenderer) tableHeader() string {
	columns := make([]string, len(r.columns))
	for i, col := range r.columns {
		columns[i] = col.title
	}
	return "<green>" + strings.Join(columns, "\t") + "</>"
}

func (r *DockerDiskUsageRenderer) formattedDiskUsage() string {
	buf := bytes.NewBufferString("")
	usage := r.diskUsage
	context := formatter.DiskUsageContext{
		Context: formatter.Context{
			Output: buf,
			Format: formatter.NewDiskUsageFormat(formatter.TableFormatKey),
		},
		LayersSize: usage.LayersSize,
		Images:     usage.Images,
		Containers: usage.Containers,
		Volumes:    usage.Volumes,
		Verbose:    false,
	}
	err := context.Write()
	if err != nil {
		return err.Error()
	}
	//The header from the table created by the formatter is removed
	duTable := buf.String()
	duTable = duTable[strings.Index(duTable, "\n"):]
	return duTable
}

func buildDiskUsageTableTemplate() *template.Template {
	markup :=
		`{{.DiskUsageTable}}

{{.PruneTable}}
`
	return template.Must(template.New(`diskUsageTable`).Parse(markup))
}
