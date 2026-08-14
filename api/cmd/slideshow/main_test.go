package main

import (
	"strings"
	"testing"
)

func filterComplexOf(t *testing.T, args []string) string {
	t.Helper()
	for i, a := range args {
		if a == "-filter_complex" {
			return args[i+1]
		}
	}
	t.Fatal("no -filter_complex in args")
	return ""
}

func TestXfadeOffsets(t *testing.T) {
	// 3 clips of 270 frames, 45-frame transitions at 30 fps:
	// offset1 = (270-45)/30 = 7.5s, offset2 = (540-90)/30 = 15s
	args := xfadeArgs([]string{"a.mp4", "b.mp4", "c.mp4"}, []int{270, 270, 270}, 45, 30, "out.mp4")
	graph := filterComplexOf(t, args)
	for _, want := range []string{"offset=7.500000", "offset=15.000000", "duration=1.500000"} {
		if !strings.Contains(graph, want) {
			t.Errorf("graph missing %q: %s", want, graph)
		}
	}
}

func TestChunkedMergeFrameInvariant(t *testing.T) {
	// Chunked merging must land on the same total as one flat xfade chain.
	const tf = 45
	frames := make([]int, 45)
	for i := range frames {
		frames[i] = 270
	}
	chunked := mergedFrames([]int{
		mergedFrames(frames[0:20], tf),
		mergedFrames(frames[20:40], tf),
		mergedFrames(frames[40:45], tf),
	}, tf)
	if flat := mergedFrames(frames, tf); chunked != flat {
		t.Errorf("chunked=%d flat=%d", chunked, flat)
	}
}

func TestIsInternal(t *testing.T) {
	img := func(names ...string) apiImage {
		var i apiImage
		for _, n := range names {
			name := n
			i.Tags = append(i.Tags, struct {
				Tag *struct {
					Name string `json:"name"`
				} `json:"tag"`
			}{Tag: &struct {
				Name string `json:"name"`
			}{Name: name}})
		}
		return i
	}
	if !isInternal(img("trip| Internal ")) {
		t.Error("combo tag part 'internal' must match")
	}
	if isInternal(img("internally", "trip")) {
		t.Error("'internally' must not match")
	}
}

func TestPickVariantsNoAdjacentRepeat(t *testing.T) {
	picked := pickVariants(200, []int{variantZoomIn, variantPanRight, variantZoomOut, variantPanLeft})
	for i := 1; i < len(picked); i++ {
		if picked[i] == picked[i-1] {
			t.Fatalf("adjacent repeat at %d: %v", i, picked[i])
		}
	}
	for _, v := range pickVariants(5, []int{variantZoomOut}) {
		if v != variantZoomOut {
			t.Fatal("single-variant set must always yield that variant")
		}
	}
}
