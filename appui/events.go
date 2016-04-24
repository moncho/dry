package appui

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/docker/docker/pkg/jsonlog"
	"github.com/docker/engine-api/types/events"
	"github.com/moncho/dry/ui"
)

type eventsRenderer struct {
	events []events.Message
}

//NewDockerEventsRenderer creates a renderer for docker events
func NewDockerEventsRenderer(events []events.Message) ui.Renderer {
	return &eventsRenderer{
		events: events,
	}
}

func (r *eventsRenderer) Render() string {
	buf := bytes.NewBufferString("")

	w := tabwriter.NewWriter(buf, 20, 1, 3, ' ', 0)
	io.WriteString(w, "\n")

	io.WriteString(w, "<blue><b>EVENTS - showing the last 10 events</></>\n\n")

	if len(r.events) == 0 {
		io.WriteString(w, "<red>Docker daemon has not reported events.</>\n\n")
	}
	for _, event := range r.events {
		printEvent(w, event)
	}
	w.Flush()
	return buf.String()
}

func printEvent(w io.Writer, event events.Message) {
	io.WriteString(w, "<white>")

	if event.TimeNano != 0 {
		fmt.Fprintf(w, "%s ", time.Unix(0, event.TimeNano).Format(jsonlog.RFC3339NanoFixed))
	} else if event.Time != 0 {
		fmt.Fprintf(w, "%s ", time.Unix(event.Time, 0).Format(jsonlog.RFC3339NanoFixed))
	}

	fmt.Fprintf(w, "</><blue>%s %s %s</><white>", event.Type, event.Action, event.Actor.ID)

	if len(event.Actor.Attributes) > 0 {
		var attrs []string
		var keys []string
		for k := range event.Actor.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := event.Actor.Attributes[k]
			attrs = append(attrs, fmt.Sprintf("%s=%s", k, v))
		}
		fmt.Fprintf(w, " (%s)", strings.Join(attrs, ", "))
	}
	fmt.Fprint(w, "</>\n\n")
}
