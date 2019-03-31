package generator

import (
	"sort"
)

// Pipe has the information needed to represent it.
type Pipe struct {
	ImportPath string
	Module     string
	Version    string
	Alias      string
}

type templateContent struct {
	Pipe            []templateContentPipe
	PipeModuleCount int
}

type templateContentPipe struct {
	ImportPath      string
	ImportPathAlias string // Alias extracted from the path.
	Alias           string // Overwrite the above path alias value.
	Revision        string
	Module          string
}

type templateContentPipeSlice []templateContentPipe

func (ps templateContentPipeSlice) Len() int {
	return len(ps)
}

func (ps templateContentPipeSlice) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

func (ps templateContentPipeSlice) Less(i, j int) bool {
	paths := make([]string, 0, 2)
	add := func(index int) {
		if ps[index].Module != "" {
			paths = append(paths, ps[index].Module)
		} else {
			paths = append(paths, ps[index].ImportPath)
		}
	}
	add(i)
	add(j)
	return sort.StringsAreSorted(paths)
}
