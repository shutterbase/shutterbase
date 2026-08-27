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
	// membership follows every face an image exposes, not just its own cluster,
	// so clicking a secondary face lands on a gallery that contains the source
	members := make([]aiserver.PersonImage, 0, len(all))
	for _, id := range all {
		if face, ok := fakeFaceOf(id, fakePersonRef(cluster)); ok {
			members = append(members, face)
		}
	}
	resp.Total = len(members)
	start := page * pageSize
	if start >= len(members) {
		return resp, nil
	}
	resp.Items = members[start:min(start+pageSize, len(members))]
	return resp, nil
}

func fakeFaceOf(imageRef, personRef string) (aiserver.PersonImage, bool) {
	for _, f := range fakeFacesFor(imageRef) {
		if f.PersonRef == personRef {
			return aiserver.PersonImage{ImageRef: imageRef, X: f.X, Y: f.Y, W: f.W, H: f.H}, true
		}
	}
	return aiserver.PersonImage{}, false
}

func (s *fakeAIRemote) Similar(_ context.Context, _, _ string, _, _ int) (aiserver.SimilarResponse, error) {
	return aiserver.SimilarResponse{Items: []aiserver.SimilarImage{}}, nil
}

// Search has no descriptions to rank in the stub; an empty page keeps the UI
// flow (dialog → "nothing matches") exercisable without fsai.
func (s *fakeAIRemote) Search(_ context.Context, _, _ string, page, pageSize int) (aiserver.SimilarResponse, error) {
	return aiserver.SimilarResponse{Items: []aiserver.SimilarImage{}, Page: page, PageSize: pageSize}, nil
}

func (s *fakeAIRemote) Persons(ctx context.Context, projectIDs []string, page, pageSize int) (aiserver.PersonsResponse, error) {
	type tally struct {
		count  int
		sample aiserver.PersonImage
	}
	counts := map[string]*tally{}
	for _, pid := range projectIDs {
		refs, err := s.projectImageRefs(ctx, pid)
		if err != nil {
			return aiserver.PersonsResponse{}, err
		}
		for _, ref := range refs {
			for _, f := range fakeFacesFor(ref) {
				t, ok := counts[f.PersonRef]
				if !ok {
					t = &tally{sample: aiserver.PersonImage{ImageRef: ref, X: f.X, Y: f.Y, W: f.W, H: f.H}}
					counts[f.PersonRef] = t
				}
				t.count++
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
		resp.Items = append(resp.Items, aiserver.PersonEntry{PersonRef: key, Count: counts[key].count, Sample: counts[key].sample})
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
