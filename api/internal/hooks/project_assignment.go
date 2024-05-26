package hooks

import (
  "github.com/shutterbase/shutterbase/internal/util"
  "github.com/pocketbase/pocketbase/core"
)

func registerProjectAssignmentHooks(context *util.Context) {
	context.App.OnRecordAfterCreateRequest("project_assignments").Add(func(e *core.RecordCreateEvent) error {

		projectAssignmentUserId := e.Record.GetString("user")
		projectAssignmentUser, err := context.App.Dao().FindRecordById("users", projectAssignmentUserId)
		if err != nil {
			return err
		}

		projectAssignmentIds := projectAssignmentUser.GetStringSlice("projectAssignments")
		projectAssignmentIds = append(projectAssignmentIds, e.Record.Id)
		projectAssignmentUser.Set("projectAssignments", projectAssignmentIds)

		if err := context.App.Dao().SaveRecord(projectAssignmentUser); err != nil {
			return err
		}

		return nil
	})

	context.App.OnRecordAfterDeleteRequest("project_assignments").Add(func(e *core.RecordDeleteEvent) error {

		projectAssignmentUserId := e.Record.GetString("user")
		projectAssignmentUser, err := context.App.Dao().FindRecordById("users", projectAssignmentUserId)
		if err != nil {
			return err
		}

		projectAssignmentIds := projectAssignmentUser.GetStringSlice("projectAssignments")
		projectAssignmentIds = util.RemoveStringFromSlice(projectAssignmentIds, e.Record.Id)
		projectAssignmentUser.Set("projectAssignments", projectAssignmentIds)

		if err := context.App.Dao().SaveRecord(projectAssignmentUser); err != nil {
			return err
		}

		return nil
	})
}