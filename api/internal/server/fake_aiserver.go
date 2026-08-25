package server

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"

	"github.com/shutterbase/shutterbase/ent"
	entimage "github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/pkg/aiserver"
)

// fakeAIRemote is a DEV-only stand-in for the pkg/aiserver remote behind the
// faces / person-search / merge proxies. It deterministically clusters every
// project's images by ID hash into fakeAIPersonCount persons, so the whole
// face pipeline (boxes, person grids, People overview) is exercisable locally
// without an external AI server. Tagging inference is unaffected — this only
// implements the proxy contract.
type fakeAIRemote struct {
	client *ent.Client
}

const fakeAIPersonCount = 3

func NewFakeAIRemote(client *ent.Client) aiserver.Server {
	return &fakeAIRemote{client: client}
}

func fakeCluster(imageRef string) int {
	h := fnv.New32a()
	h.Write([]byte(imageRef))
	return int(h.Sum32() % fakeAIPersonCount)
}

func fakePersonRef(cluster int) string {
	return fmt.Sprintf("dev-person-%d", cluster)
}

func parseFakePersonRef(personRef string) (int, bool) {
	raw, ok := strings.CutPrefix(personRef, "dev-person-")
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(raw)
	if err != nil || i < 0 || i >= fakeAIPersonCount {
		return 0, false
	}
	return i, true
}

func fakeFaceBox(imageRef string, faceIdx int) (x, y, w, boxH float64) {
	h := fnv.New32a()
	fmt.Fprintf(h, "%s:%d", imageRef, faceIdx)
	v := h.Sum32()
	x = 0.12 + float64(v%48)/100
	y = 0.08 + float64((v>>6)%42)/100
	w = 0.16 + float64((v>>13)%14)/100
	boxH = 0.20 + float64((v>>19)%10)/100
	return x, y, w, boxH
}

func fakeFacesFor(imageRef string) []aiserver.Face {
	cluster := fakeCluster(imageRef)
	var own aiserver.Face
	own.PersonRef = fakePersonRef(cluster)
	own.X, own.Y, own.W, own.H = fakeFaceBox(imageRef, 0)
	faces := []aiserver.Face{own}
	h := fnv.New32a()
	h.Write([]byte(imageRef))
	if h.Sum32()%2 == 0 {
		var neighbor aiserver.Face
		neighbor.PersonRef = fakePersonRef((cluster + 1) % fakeAIPersonCount)
		neighbor.X, neighbor.Y, neighbor.W, neighbor.H = fakeFaceBox(imageRef, 1)
		faces = append(faces, neighbor)
	}
	return faces
}

func (s *fakeAIRemote) projectImageRefs(ctx context.Context, projectID string) ([]string, error) {
	return s.client.Image.Query().Where(entimage.ProjectID(projectID)).Order(entimage.ByID()).IDs(ctx)
}

func (s *fakeAIRemote) Prime(_ context.Context, _ string, _ aiserver.Project) error { return nil }

func (s *fakeAIRemote) Ingest(_ context.Context, _ string, req aiserver.IngestRequest) (aiserver.IngestResponse, error) {
	return aiserver.IngestResponse{ImageRef: req.ImageRef}, nil
}

func (s *fakeAIRemote) Faces(ctx context.Context, projectID, imageRef string) (aiserver.FacesResponse, error) {
	img, err := s.client.Image.Get(ctx, imageRef)
	if err != nil || img.ProjectID != projectID {
		return aiserver.FacesResponse{}, aiserver.ErrNotFound
	}
	return aiserver.FacesResponse{ImageRef: imageRef, Faces: fakeFacesFor(imageRef)}, nil
}

func (s *fakeAIRemote) PersonImages(ctx context.Context, projectID, personRef string, page, pageSize int, _ bool) (aiserver.PersonImagesResponse, error) {
	resp := aiserver.PersonImagesResponse{Items: []aiserver.PersonImage{}, Page: page, PageSize: pageSize}
	cluster, ok := parseFakePersonRef(personRef)
	if !ok || pageSize <= 0 {
		return resp, nil
	}
	all, err := s.projectImageRefs(ctx, projectID)
	if err != nil {
		return resp, err
	}
	members := make([]string, 0, len(all))
	for _, id := range all {
		if fakeCluster(id) == cluster {
			members = append(members, id)
		}
	}
	resp.Total = len(members)
	start := page * pageSize
	if start >= len(members) {
		return resp, nil
	}
	end := min(start+pageSize, len(members))
	for _, id := range members[start:end] {
		x, y, w, hh := fakeFaceBox(id, 0)
		resp.Items = append(resp.Items, aiserver.PersonImage{ImageRef: id, X: x, Y: y, W: w, H: hh})
	}
	return resp, nil
}

func (s *fakeAIRemote) Similar(_ context.Context, _, _ string, _, _ int) (aiserver.SimilarResponse, error) {
	return aiserver.SimilarResponse{Items: []aiserver.SimilarImage{}}, nil
}

func (s *fakeAIRemote) Persons(ctx context.Context, projectIDs []string, page, pageSize int) (aiserver.PersonsResponse, error) {
	type tally struct {
		count  int
		sample string
	}
	counts := map[string]*tally{}
	for _, pid := range projectIDs {
		refs, err := s.projectImageRefs(ctx, pid)
		if err != nil {
			return aiserver.PersonsResponse{}, err
		}
		for _, ref := range refs {
			key := fakePersonRef(fakeCluster(ref))
			t, ok := counts[key]
			if !ok {
				t = &tally{}
				counts[key] = t
			}
			t.count++
			if t.sample == "" {
				t.sample = ref
			}
		}
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		ci, cj := counts[keys[i]].count, counts[keys[j]].count
		if ci != cj {
			return ci > cj
		}
		return keys[i] < keys[j]
	})
	resp := aiserver.PersonsResponse{Items: []aiserver.PersonEntry{}, Total: len(keys), Page: page, PageSize: pageSize}
	start := page * pageSize
	if pageSize <= 0 || start >= len(keys) {
		return resp, nil
	}
	end := min(start+pageSize, len(keys))
	for _, key := range keys[start:end] {
		t := counts[key]
		x, y, w, hh := fakeFaceBox(t.sample, 0)
		resp.Items = append(resp.Items, aiserver.PersonEntry{
			PersonRef: key,
			Count:     t.count,
			Sample:    aiserver.PersonImage{ImageRef: t.sample, X: x, Y: y, W: w, H: hh},
		})
	}
	return resp, nil
}

func (s *fakeAIRemote) MergeCandidates(_ context.Context, _ []string, _ int, _ string) (aiserver.MergeCandidatesResponse, error) {
	return aiserver.MergeCandidatesResponse{}, nil
}

func (s *fakeAIRemote) DecideMerge(_ context.Context, _ aiserver.MergeDecision) error { return nil }

func (s *fakeAIRemote) Merges(_ context.Context, _ []string) (aiserver.MergesResponse, error) {
	return aiserver.MergesResponse{Items: []aiserver.Merge{}}, nil
}

func (s *fakeAIRemote) DeleteMerge(_ context.Context, _, _ string) error {
	return errors.New("fake AI server has no merges")
}

func (s *fakeAIRemote) Recluster(_ context.Context, _ string) error { return nil }

func (s *fakeAIRemote) DeleteImage(_ context.Context, _, _ string) error { return nil }
