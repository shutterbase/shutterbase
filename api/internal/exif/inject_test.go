package exif

import (
	"reflect"
	"testing"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
)

func tag(name string, tagType imagetag.Type, order *int) *ent.ImageTag {
	return &ent.ImageTag{Name: name, Type: tagType, Order: order}
}

func intPtr(v int) *int { return &v }

func TestSortTagsByOrder(t *testing.T) {
	tags := []*ent.ImageTag{
		tag("zebra", imagetag.TypeManual, nil),
		tag("bravo", imagetag.TypeManual, intPtr(2)),
		tag("delta", imagetag.TypeManual, intPtr(1)),
		tag("alpha", imagetag.TypeManual, nil),
		tag("charlie", imagetag.TypeManual, intPtr(2)),
	}
	got := []string{}
	for _, tg := range sortTagsByOrder(tags) {
		got = append(got, tg.Name)
	}
	// ranked ascending, order ties alphabetical, unset last alphabetical
	want := []string{"delta", "bravo", "charlie", "alpha", "zebra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBuildMetadataKeywordsRespectOrder(t *testing.T) {
	image := &ent.Image{
		Edges: ent.ImageEdges{
			ImageTagAssignments: []*ent.ImageTagAssignment{
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("manual-late", imagetag.TypeManual, nil)}},
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("default-first", imagetag.TypeDefault, intPtr(1))}},
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("custom-hidden", imagetag.TypeCustom, intPtr(1))}},
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("internal", imagetag.TypeDefault, intPtr(1))}},
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("manual-second", imagetag.TypeManual, intPtr(5))}},
			},
		},
	}
	m := buildMetadata(image)
	want := []string{"default-first", "manual-second", "manual-late"}
	if !reflect.DeepEqual(m["EXIF:XPKeywords"], want) {
		t.Fatalf("XPKeywords = %v, want %v", m["EXIF:XPKeywords"], want)
	}
	if !reflect.DeepEqual(m["IPTC:Keywords"], want) {
		t.Fatalf("IPTC:Keywords = %v, want %v", m["IPTC:Keywords"], want)
	}
}

func TestBuildMetadataComboTagsSplitAtExport(t *testing.T) {
	image := &ent.Image{
		Edges: ent.ImageEdges{
			ImageTagAssignments: []*ent.ImageTagAssignment{
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("autocross|DV", imagetag.TypeManual, intPtr(1))}},
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("autocross", imagetag.TypeManual, intPtr(2))}},
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag(" a | b ||c ", imagetag.TypeManual, intPtr(3))}},
			},
		},
	}
	m := buildMetadata(image)
	// combo split in tag order, duplicate "autocross" deduped, parts trimmed, empties dropped
	want := []string{"autocross", "DV", "a", "b", "c"}
	if !reflect.DeepEqual(m["IPTC:Keywords"], want) {
		t.Fatalf("IPTC:Keywords = %v, want %v", m["IPTC:Keywords"], want)
	}
	if !reflect.DeepEqual(m["EXIF:XPKeywords"], want) {
		t.Fatalf("EXIF:XPKeywords = %v, want %v", m["EXIF:XPKeywords"], want)
	}
}

func TestBuildMetadataComboTagCannotSmuggleInternal(t *testing.T) {
	image := &ent.Image{
		Edges: ent.ImageEdges{
			ImageTagAssignments: []*ent.ImageTagAssignment{
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("trip|internal", imagetag.TypeManual, intPtr(1))}},
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("internal", imagetag.TypeDefault, intPtr(2))}},
			},
		},
	}
	m := buildMetadata(image)
	want := []string{"trip"}
	if !reflect.DeepEqual(m["IPTC:Keywords"], want) {
		t.Fatalf("IPTC:Keywords = %v, want %v (reserved internal tag must never export)", m["IPTC:Keywords"], want)
	}
}

func TestBuildMetadataComboTagPartGetsCopyrightPrefix(t *testing.T) {
	image := &ent.Image{
		Edges: ent.ImageEdges{
			User:    &ent.User{CopyrightTag: "mm"},
			Project: &ent.Project{CopyrightTagPrefix: "by_"},
			ImageTagAssignments: []*ent.ImageTagAssignment{
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("autocross|mm", imagetag.TypeManual, intPtr(1))}},
			},
		},
	}
	m := buildMetadata(image)
	want := []string{"autocross", "by_mm"}
	if !reflect.DeepEqual(m["IPTC:Keywords"], want) {
		t.Fatalf("IPTC:Keywords = %v, want %v", m["IPTC:Keywords"], want)
	}
}

func TestBuildMetadataCopyrightTagPrefix(t *testing.T) {
	image := &ent.Image{
		Edges: ent.ImageEdges{
			User:    &ent.User{FirstName: "Max", LastName: "Mustermann", CopyrightTag: "mm"},
			Project: &ent.Project{CopyrightTagPrefix: "by_"},
			ImageTagAssignments: []*ent.ImageTagAssignment{
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("mm", imagetag.TypeDefault, intPtr(1))}},
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("autocross", imagetag.TypeManual, intPtr(2))}},
			},
		},
	}
	m := buildMetadata(image)
	want := []string{"by_mm", "autocross"}
	if !reflect.DeepEqual(m["IPTC:Keywords"], want) {
		t.Fatalf("IPTC:Keywords = %v, want %v", m["IPTC:Keywords"], want)
	}
	if m["IPTC:By-lineTitle"] != "by_mm" {
		t.Fatalf("By-lineTitle = %v, want by_mm", m["IPTC:By-lineTitle"])
	}
}

func TestBuildMetadataPrefixWithEmptyCopyrightTag(t *testing.T) {
	image := &ent.Image{
		Edges: ent.ImageEdges{
			User:    &ent.User{CopyrightTag: ""},
			Project: &ent.Project{CopyrightTagPrefix: "by_"},
			ImageTagAssignments: []*ent.ImageTagAssignment{
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("autocross", imagetag.TypeManual, intPtr(1))}},
			},
		},
	}
	m := buildMetadata(image)
	if !reflect.DeepEqual(m["IPTC:Keywords"], []string{"autocross"}) {
		t.Fatalf("IPTC:Keywords = %v, want [autocross]", m["IPTC:Keywords"])
	}
	if m["IPTC:By-lineTitle"] != "" {
		t.Fatalf("By-lineTitle = %v, want empty (a bare prefix must never render)", m["IPTC:By-lineTitle"])
	}
}

func TestBuildMetadataPrefixWithoutEdges(t *testing.T) {
	image := &ent.Image{
		Edges: ent.ImageEdges{
			ImageTagAssignments: []*ent.ImageTagAssignment{
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("autocross", imagetag.TypeManual, intPtr(1))}},
			},
		},
	}
	m := buildMetadata(image)
	if !reflect.DeepEqual(m["IPTC:Keywords"], []string{"autocross"}) {
		t.Fatalf("IPTC:Keywords = %v, want [autocross]", m["IPTC:Keywords"])
	}
	if _, ok := m["IPTC:By-lineTitle"]; ok {
		t.Fatal("By-lineTitle must be absent without a User edge")
	}
}

func TestBuildMetadataNoPrefixLeavesTagsUntouched(t *testing.T) {
	image := &ent.Image{
		Edges: ent.ImageEdges{
			User:    &ent.User{CopyrightTag: "mm"},
			Project: &ent.Project{},
			ImageTagAssignments: []*ent.ImageTagAssignment{
				{Edges: ent.ImageTagAssignmentEdges{ImageTag: tag("mm", imagetag.TypeDefault, intPtr(1))}},
			},
		},
	}
	m := buildMetadata(image)
	if !reflect.DeepEqual(m["IPTC:Keywords"], []string{"mm"}) {
		t.Fatalf("IPTC:Keywords = %v, want [mm]", m["IPTC:Keywords"])
	}
	if m["IPTC:By-lineTitle"] != "mm" {
		t.Fatalf("By-lineTitle = %v, want mm", m["IPTC:By-lineTitle"])
	}
}
