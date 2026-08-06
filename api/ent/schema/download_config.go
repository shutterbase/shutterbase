package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DownloadConfig is a user's saved bulk-download preset for one project: tag
// filters, per-image blocklist and folder options for the in-browser download
// page (the successor of cmd/downloader). lastDownloadAt is the server-side
// replacement of the CLI's .timestamp file: the start time of the last
// completed run, which the client uses as the delta window (minus a safety
// margin) on the next run.
type DownloadConfig struct{ ent.Schema }

func (DownloadConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{StringIDMixin{}, AuditMixin{}}
}

func (DownloadConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().StructTag(`json:"name"`),
		// whitelist tags are AND-applied server-side by /images; blacklist tags
		// and blocked images are excluded client-side by the runner.
		field.JSON("whitelistTagIds", []string{}).Optional().Default([]string{}).StructTag(`json:"whitelistTagIds"`),
		field.JSON("blacklistTagIds", []string{}).Optional().Default([]string{}).StructTag(`json:"blacklistTagIds"`),
		field.JSON("blockedImageIds", []string{}).Optional().Default([]string{}).StructTag(`json:"blockedImageIds"`),
		// deltaSubfolder writes new/changed files into a per-run delta_<date>
		// subfolder (issue #26); groupByDate sorts into capture-date folders
		// (PR #40 "upload" mode).
		field.Bool("deltaSubfolder").Default(false).StructTag(`json:"deltaSubfolder"`),
		field.Bool("groupByDate").Default(false).StructTag(`json:"groupByDate"`),
		field.String("folder_structure").Default("default").StructTag(`json:"folderStructure"`),
		field.Time("lastDownloadAt").Optional().Nillable().StructTag(`json:"lastDownloadAt"`),
		field.String("project_id").StructTag(`json:"-"`),
		field.UUID("user_id", uuid.UUID{}).StructTag(`json:"-"`),
	}
}

func (DownloadConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("downloadConfigs").Field("project_id").Unique().Required(),
		edge.From("user", User.Type).Ref("downloadConfigs").Field("user_id").Unique().Required(),
	}
}

func (DownloadConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("user_id", "project_id"),
	}
}