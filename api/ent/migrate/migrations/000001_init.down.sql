-- reverse: create index "timeoffset_camera_id" to table: "time_offsets"
DROP INDEX "timeoffset_camera_id";
-- reverse: create "time_offsets" table
DROP TABLE "time_offsets";
-- reverse: create index "projectassignment_user_id" to table: "project_assignments"
DROP INDEX "projectassignment_user_id";
-- reverse: create index "projectassignment_role_id" to table: "project_assignments"
DROP INDEX "projectassignment_role_id";
-- reverse: create index "projectassignment_project_id_user_id" to table: "project_assignments"
DROP INDEX "projectassignment_project_id_user_id";
-- reverse: create "project_assignments" table
DROP TABLE "project_assignments";
-- reverse: create index "roles_key_key" to table: "roles"
DROP INDEX "roles_key_key";
-- reverse: create "roles" table
DROP TABLE "roles";
-- reverse: create index "imagetagassignment_image_tag_id" to table: "image_tag_assignments"
DROP INDEX "imagetagassignment_image_tag_id";
-- reverse: create index "imagetagassignment_image_id_image_tag_id" to table: "image_tag_assignments"
DROP INDEX "imagetagassignment_image_id_image_tag_id";
-- reverse: create index "imagetagassignment_image_id" to table: "image_tag_assignments"
DROP INDEX "imagetagassignment_image_id";
-- reverse: create "image_tag_assignments" table
DROP TABLE "image_tag_assignments";
-- reverse: create index "images_storage_id_key" to table: "images"
DROP INDEX "images_storage_id_key";
-- reverse: create index "images_computed_file_name_key" to table: "images"
DROP INDEX "images_computed_file_name_key";
-- reverse: create index "image_user_id" to table: "images"
DROP INDEX "image_user_id";
-- reverse: create index "image_upload_id" to table: "images"
DROP INDEX "image_upload_id";
-- reverse: create index "image_project_id_captured_at_corrected" to table: "images"
DROP INDEX "image_project_id_captured_at_corrected";
-- reverse: create index "image_project_id" to table: "images"
DROP INDEX "image_project_id";
-- reverse: create index "image_image_tags" to table: "images"
DROP INDEX "image_image_tags";
-- reverse: create index "image_captured_at_corrected" to table: "images"
DROP INDEX "image_captured_at_corrected";
-- reverse: create index "image_camera_id" to table: "images"
DROP INDEX "image_camera_id";
-- reverse: create "images" table
DROP TABLE "images";
-- reverse: create index "upload_user_id" to table: "uploads"
DROP INDEX "upload_user_id";
-- reverse: create index "upload_project_id" to table: "uploads"
DROP INDEX "upload_project_id";
-- reverse: create index "upload_camera_id" to table: "uploads"
DROP INDEX "upload_camera_id";
-- reverse: create "uploads" table
DROP TABLE "uploads";
-- reverse: create index "imagetag_project_id" to table: "image_tags"
DROP INDEX "imagetag_project_id";
-- reverse: create index "imagetag_name_project_id" to table: "image_tags"
DROP INDEX "imagetag_name_project_id";
-- reverse: create "image_tags" table
DROP TABLE "image_tags";
-- reverse: create index "camera_user_id" to table: "cameras"
DROP INDEX "camera_user_id";
-- reverse: create index "camera_name_user_id" to table: "cameras"
DROP INDEX "camera_name_user_id";
-- reverse: create "cameras" table
DROP TABLE "cameras";
-- reverse: create index "users_username_key" to table: "users"
DROP INDEX "users_username_key";
-- reverse: create index "users_legacy_id_key" to table: "users"
DROP INDEX "users_legacy_id_key";
-- reverse: create index "users_email_key" to table: "users"
DROP INDEX "users_email_key";
-- reverse: create index "user_first_name_last_name" to table: "users"
DROP INDEX "user_first_name_last_name";
-- reverse: create index "user_active_project_id" to table: "users"
DROP INDEX "user_active_project_id";
-- reverse: create "users" table
DROP TABLE "users";
-- reverse: create index "projects_name_key" to table: "projects"
DROP INDEX "projects_name_key";
-- reverse: create "projects" table
DROP TABLE "projects";
