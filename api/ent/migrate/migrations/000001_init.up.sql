-- create "projects" table
CREATE TABLE "projects" ("id" character varying NOT NULL, "createdAt" timestamptz NOT NULL, "updatedAt" timestamptz NOT NULL, "created_by" uuid NULL, "updated_by" uuid NULL, "name" character varying NOT NULL, "description" character varying NOT NULL, "copyright" character varying NOT NULL, "copyright_reference" character varying NOT NULL, "location_name" character varying NOT NULL, "location_code" character varying NOT NULL, "location_city" character varying NOT NULL, "ai_system_message" character varying NULL, PRIMARY KEY ("id"));
-- create index "projects_name_key" to table: "projects"
CREATE UNIQUE INDEX "projects_name_key" ON "projects" ("name");
-- create "users" table
CREATE TABLE "users" ("id" uuid NOT NULL, "createdAt" timestamptz NOT NULL, "updatedAt" timestamptz NOT NULL, "created_by" uuid NULL, "updated_by" uuid NULL, "legacy_id" character varying NULL, "username" character varying NOT NULL, "first_name" character varying NOT NULL, "last_name" character varying NOT NULL, "copyright_tag" character varying NULL, "active" boolean NOT NULL DEFAULT false, "email" character varying NULL, "verified" boolean NOT NULL DEFAULT false, "password_hash" character varying NULL, "force_password_change" boolean NOT NULL DEFAULT false, "provider" character varying NOT NULL DEFAULT 'local', "role" character varying NOT NULL DEFAULT 'user', "active_project_id" character varying NULL, PRIMARY KEY ("id"), CONSTRAINT "users_projects_activeProject" FOREIGN KEY ("active_project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE SET NULL);
-- create index "user_active_project_id" to table: "users"
CREATE INDEX "user_active_project_id" ON "users" ("active_project_id");
-- create index "user_first_name_last_name" to table: "users"
CREATE UNIQUE INDEX "user_first_name_last_name" ON "users" ("first_name", "last_name");
-- create index "users_email_key" to table: "users"
CREATE UNIQUE INDEX "users_email_key" ON "users" ("email");
-- create index "users_legacy_id_key" to table: "users"
CREATE UNIQUE INDEX "users_legacy_id_key" ON "users" ("legacy_id");
-- create index "users_username_key" to table: "users"
CREATE UNIQUE INDEX "users_username_key" ON "users" ("username");
-- create "cameras" table
CREATE TABLE "cameras" ("id" character varying NOT NULL, "createdAt" timestamptz NOT NULL, "updatedAt" timestamptz NOT NULL, "created_by" uuid NULL, "updated_by" uuid NULL, "name" character varying NOT NULL, "user_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "cameras_users_cameras" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- create index "camera_name_user_id" to table: "cameras"
CREATE UNIQUE INDEX "camera_name_user_id" ON "cameras" ("name", "user_id");
-- create index "camera_user_id" to table: "cameras"
CREATE INDEX "camera_user_id" ON "cameras" ("user_id");
-- create "image_tags" table
CREATE TABLE "image_tags" ("id" character varying NOT NULL, "createdAt" timestamptz NOT NULL, "updatedAt" timestamptz NOT NULL, "created_by" uuid NULL, "updated_by" uuid NULL, "name" character varying NOT NULL, "description" character varying NOT NULL, "is_album" boolean NOT NULL DEFAULT false, "type" character varying NOT NULL, "project_id" character varying NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "image_tags_projects_imageTags" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- create index "imagetag_name_project_id" to table: "image_tags"
CREATE UNIQUE INDEX "imagetag_name_project_id" ON "image_tags" ("name", "project_id");
-- create index "imagetag_project_id" to table: "image_tags"
CREATE INDEX "imagetag_project_id" ON "image_tags" ("project_id");
-- create "uploads" table
CREATE TABLE "uploads" ("id" character varying NOT NULL, "createdAt" timestamptz NOT NULL, "updatedAt" timestamptz NOT NULL, "created_by" uuid NULL, "updated_by" uuid NULL, "name" character varying NOT NULL, "camera_id" character varying NOT NULL, "project_id" character varying NOT NULL, "user_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "uploads_cameras_uploads" FOREIGN KEY ("camera_id") REFERENCES "cameras" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "uploads_projects_uploads" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "uploads_users_uploads" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- create index "upload_camera_id" to table: "uploads"
CREATE INDEX "upload_camera_id" ON "uploads" ("camera_id");
-- create index "upload_project_id" to table: "uploads"
CREATE INDEX "upload_project_id" ON "uploads" ("project_id");
-- create index "upload_user_id" to table: "uploads"
CREATE INDEX "upload_user_id" ON "uploads" ("user_id");
-- create "images" table
CREATE TABLE "images" ("id" character varying NOT NULL, "createdAt" timestamptz NOT NULL, "updatedAt" timestamptz NOT NULL, "created_by" uuid NULL, "updated_by" uuid NULL, "file_name" character varying NOT NULL, "computed_file_name" character varying NULL, "storage_id" character varying NOT NULL, "exif_data" jsonb NULL, "image_tags" jsonb NULL, "captured_at" timestamptz NULL, "captured_at_corrected" timestamptz NULL, "inferred_at" timestamptz NULL, "size" bigint NOT NULL, "width" bigint NULL, "height" bigint NULL, "camera_id" character varying NOT NULL, "project_id" character varying NOT NULL, "upload_id" character varying NOT NULL, "user_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "images_cameras_images" FOREIGN KEY ("camera_id") REFERENCES "cameras" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "images_projects_images" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "images_uploads_images" FOREIGN KEY ("upload_id") REFERENCES "uploads" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "images_users_images" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- create index "image_camera_id" to table: "images"
CREATE INDEX "image_camera_id" ON "images" ("camera_id");
-- create index "image_captured_at_corrected" to table: "images"
CREATE INDEX "image_captured_at_corrected" ON "images" ("captured_at_corrected");
-- create index "image_image_tags" to table: "images"
CREATE INDEX "image_image_tags" ON "images" USING gin ("image_tags" jsonb_path_ops);
-- create index "image_project_id" to table: "images"
CREATE INDEX "image_project_id" ON "images" ("project_id");
-- create index "image_project_id_captured_at_corrected" to table: "images"
CREATE INDEX "image_project_id_captured_at_corrected" ON "images" ("project_id", "captured_at_corrected");
-- create index "image_upload_id" to table: "images"
CREATE INDEX "image_upload_id" ON "images" ("upload_id");
-- create index "image_user_id" to table: "images"
CREATE INDEX "image_user_id" ON "images" ("user_id");
-- create index "images_computed_file_name_key" to table: "images"
CREATE UNIQUE INDEX "images_computed_file_name_key" ON "images" ("computed_file_name");
-- create index "images_storage_id_key" to table: "images"
CREATE UNIQUE INDEX "images_storage_id_key" ON "images" ("storage_id");
-- create "image_tag_assignments" table
CREATE TABLE "image_tag_assignments" ("id" character varying NOT NULL, "createdAt" timestamptz NOT NULL, "updatedAt" timestamptz NOT NULL, "created_by" uuid NULL, "updated_by" uuid NULL, "type" character varying NOT NULL, "image_id" character varying NOT NULL, "image_tag_id" character varying NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "image_tag_assignments_image_tags_tagAssignments" FOREIGN KEY ("image_tag_id") REFERENCES "image_tags" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "image_tag_assignments_images_imageTagAssignments" FOREIGN KEY ("image_id") REFERENCES "images" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- create index "imagetagassignment_image_id" to table: "image_tag_assignments"
CREATE INDEX "imagetagassignment_image_id" ON "image_tag_assignments" ("image_id");
-- create index "imagetagassignment_image_id_image_tag_id" to table: "image_tag_assignments"
CREATE UNIQUE INDEX "imagetagassignment_image_id_image_tag_id" ON "image_tag_assignments" ("image_id", "image_tag_id");
-- create index "imagetagassignment_image_tag_id" to table: "image_tag_assignments"
CREATE INDEX "imagetagassignment_image_tag_id" ON "image_tag_assignments" ("image_tag_id");
-- create "roles" table
CREATE TABLE "roles" ("id" character varying NOT NULL, "createdAt" timestamptz NOT NULL, "updatedAt" timestamptz NOT NULL, "created_by" uuid NULL, "updated_by" uuid NULL, "key" character varying NOT NULL, "description" character varying NOT NULL, PRIMARY KEY ("id"));
-- create index "roles_key_key" to table: "roles"
CREATE UNIQUE INDEX "roles_key_key" ON "roles" ("key");
-- create "project_assignments" table
CREATE TABLE "project_assignments" ("id" character varying NOT NULL, "createdAt" timestamptz NOT NULL, "updatedAt" timestamptz NOT NULL, "created_by" uuid NULL, "updated_by" uuid NULL, "project_id" character varying NOT NULL, "role_id" character varying NOT NULL, "user_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "project_assignments_projects_projectAssignments" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "project_assignments_roles_projectAssignments" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "project_assignments_users_projectAssignments" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- create index "projectassignment_project_id_user_id" to table: "project_assignments"
CREATE UNIQUE INDEX "projectassignment_project_id_user_id" ON "project_assignments" ("project_id", "user_id");
-- create index "projectassignment_role_id" to table: "project_assignments"
CREATE INDEX "projectassignment_role_id" ON "project_assignments" ("role_id");
-- create index "projectassignment_user_id" to table: "project_assignments"
CREATE INDEX "projectassignment_user_id" ON "project_assignments" ("user_id");
-- create "time_offsets" table
CREATE TABLE "time_offsets" ("id" character varying NOT NULL, "createdAt" timestamptz NOT NULL, "updatedAt" timestamptz NOT NULL, "created_by" uuid NULL, "updated_by" uuid NULL, "server_time" timestamptz NOT NULL, "camera_time" timestamptz NOT NULL, "time_offset" bigint NULL, "camera_id" character varying NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "time_offsets_cameras_timeOffsets" FOREIGN KEY ("camera_id") REFERENCES "cameras" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- create index "timeoffset_camera_id" to table: "time_offsets"
CREATE INDEX "timeoffset_camera_id" ON "time_offsets" ("camera_id");
