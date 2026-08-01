package repository

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent/personname"
	"github.com/shutterbase/shutterbase/internal/util"
)

// GetPersonNames returns the known names for the given person refs; refs
// without a name are absent from the map.
func (r *Repository) GetPersonNames(ctx context.Context, refs []string) (map[string]string, error) {
	if len(refs) == 0 {
		return map[string]string{}, nil
	}
	rows, err := r.Client.PersonName.Query().Where(personname.PersonRefIn(refs...)).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting person names")
		return nil, err
	}
	names := make(map[string]string, len(rows))
	for _, row := range rows {
		names[row.PersonRef] = row.Name
	}
	return names, nil
}

// SetPersonNames names every given ref (a merge group shares one name so it
// survives merging and re-representation); an empty name clears them instead.
func (r *Repository) SetPersonNames(ctx context.Context, refs []string, name string) error {
	if len(refs) == 0 {
		return nil
	}
	if name == "" {
		_, err := r.Client.PersonName.Delete().Where(personname.PersonRefIn(refs...)).Exec(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error clearing person names")
		}
		return err
	}
	actor := util.GetActorID(ctx)
	for _, ref := range refs {
		updated, err := r.Client.PersonName.Update().
			Where(personname.PersonRefEQ(ref)).
			SetName(name).SetUpdatedBy(actor).
			Save(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error updating person name")
			return err
		}
		if updated > 0 {
			continue
		}
		_, err = r.Client.PersonName.Create().
			SetPersonRef(ref).SetName(name).
			SetCreatedBy(actor).SetUpdatedBy(actor).
			Save(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error creating person name")
			return err
		}
	}
	return nil
}
