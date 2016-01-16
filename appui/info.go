package appui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	godocker "github.com/fsouza/go-dockerclient"
	"github.com/moncho/dry/ui"
)

type infoRenderer struct {
	env *godocker.Env
}

//NewDockerInfoRenderer creates renderer for for docker info
func NewDockerInfoRenderer(env *godocker.Env) ui.Renderer {
	return &infoRenderer{
		env: env,
	}
}

//Render low-level information on a container
func (r *infoRenderer) Render() string {
	var buffer bytes.Buffer
	var keys []string
	info := r.env.Map()
	for k := range info {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := info[key]
		buffer.WriteString(fmt.Sprintf("<white> %s </>: %s\n", key, value))
	}
	return buffer.String()
}

func writeJSON(key string, json map[string]interface{}, buffer bytes.Buffer) {
	buffer.WriteString(key + ":\n")
	for k, v := range json {
		buffer.WriteString(fmt.Sprintf("<white>%s</>: %s\n", k, v))
	}
}

func asJSON(s string) (map[string]interface{}, error) {
	var js map[string]interface{}
	err := json.Unmarshal([]byte(s), &js)

	if err == nil {
		return js, nil
	}
	return nil, err
}
