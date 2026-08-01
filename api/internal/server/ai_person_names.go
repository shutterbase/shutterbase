// Person naming: user-given names for AI-server person clusters. Names live
// in the shutterbase DB (the aiserver contract is domain-agnostic and knows no
// names), keyed by personRef. Because ranked person entries surface a merge
// group under one representative ref, a name is propagated to EVERY member of
// the ref's merge group at write time — so it survives merging, unmerging and
// re-representation without any group resolution on the read path.
package server

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/pkg/aiserver"
)

// aiNamesBatchCap bounds one names lookup (a People page is 24 cards).
const aiNamesBatchCap = 200

// mergeGroup returns the connected component of ref over the merge entries
// (including ref itself), sorted for determinism.
func mergeGroup(merges []aiserver.Merge, ref string) []string {
	adjacent := map[string][]string{}
	for _, m := range merges {
		adjacent[m.PersonA] = append(adjacent[m.PersonA], m.PersonB)
		adjacent[m.PersonB] = append(adjacent[m.PersonB], m.PersonA)
	}
	seen := map[string]bool{ref: true}
	queue := []string{ref}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacent[current] {
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	group := make([]string, 0, len(seen))
	for r := range seen {
		group = append(group, r)
	}
	sort.Strings(group)
	return group
}

// aiPersonNames serves the names of the queried refs (?ref=a&ref=b) — display
// data, available to every authenticated user.
func (s *Server) aiPersonNames(c *gin.Context) {
	refs := c.QueryArray("ref")
	if len(refs) == 0 || len(refs) > aiNamesBatchCap {
		apiError(c, http.StatusBadRequest, "invalid_refs", "between 1 and 200 ref parameters required")
		return
	}
	names, err := s.Repository.GetPersonNames(c.Request.Context(), refs)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"names": names})
}

// aiSetPersonName names (or, with an empty name, un-names) a person cluster
// and its whole merge group. Same gate as merge review: curation is a
// projectAdmin's job.
func (s *Server) aiSetPersonName(c *gin.Context) {
	remote, ok := s.remote(c)
	if !ok {
		return
	}
	scope, ok := s.mergeScope(c)
	if !ok {
		return
	}
	ref := c.Param("personRef")
	var body struct {
		Name string `json:"name"`
	}
	if !bindJSON(c, &body) {
		return
	}
	name := strings.TrimSpace(body.Name)
	if len(name) > 100 {
		apiError(c, http.StatusBadRequest, "name_too_long", "person names are capped at 100 characters")
		return
	}
	ctx := c.Request.Context()
	group := []string{ref}
	if resp, err := remote.Merges(ctx, scope); err == nil {
		group = mergeGroup(resp.Items, ref)
	} else {
		// best-effort: name at least the ref itself when the AI server is down
		log.Warn().Err(err).Msg("person naming: merge group unresolved, naming the single ref")
	}
	if err := s.Repository.SetPersonNames(ctx, group, name); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"name": name})
}

// propagateMergedName spreads a merged group's single name to every member
// after a "same" verdict, so the group keeps its name whichever ref ends up
// representing it. With conflicting names it does nothing — the SPA asks the
// reviewer for the final name and PUTs it. Best-effort: a failure only logs.
func (s *Server) propagateMergedName(c *gin.Context, remote aiserver.Server, scope []string, ref string) {
	ctx := c.Request.Context()
	resp, err := remote.Merges(ctx, scope)
	if err != nil {
		log.Warn().Err(err).Msg("person naming: merge list unavailable, name not propagated")
		return
	}
	group := mergeGroup(resp.Items, ref)
	names, err := s.Repository.GetPersonNames(ctx, group)
	if err != nil {
		return
	}
	distinct := map[string]bool{}
	for _, n := range names {
		distinct[n] = true
	}
	if len(distinct) != 1 {
		return
	}
	var name string
	for n := range distinct {
		name = n
	}
	if err := s.Repository.SetPersonNames(ctx, group, name); err != nil {
		log.Warn().Err(err).Msg("person naming: name not propagated to merged group")
	}
}
