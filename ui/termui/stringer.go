package termui

import (
	"image"
	"sort"
	"strings"

	"github.com/gizak/termui"
)

type bufferer interface {
	Buffer() termui.Buffer
}

// String returns this Buffer content as a string
func String(b bufferer) (string, error) {
	cellMap := b.Buffer().CellMap
	var builder strings.Builder
	curLine := 0
	for _, k := range sortedKeys(cellMap) {
		if curLine != k.Y {
			builder.WriteByte('\n')
			curLine = k.Y
		}

		_, err := builder.WriteRune(cellMap[k].Ch)
		if err != nil {
			return "", err
		}
	}
	return builder.String(), nil
}

func sortedKeys(m map[image.Point]termui.Cell) []image.Point {
	keys := make([]image.Point, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Y == keys[j].Y {
			return keys[i].X < keys[j].X
		}
		return keys[i].Y < keys[j].Y
	})
	return keys
}
