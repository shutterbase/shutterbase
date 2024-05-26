package hooks

import "github.com/shutterbase/shutterbase/internal/util"

func RegisterHooks(context *util.Context) error {
  registerProjectAssignmentHooks(context)
  registerUserHooks(context)
  return nil
}