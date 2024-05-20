package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `[
			{
				"id": "_pb_users_auth_",
				"created": "2024-02-17 12:03:37.846Z",
				"updated": "2024-02-19 20:29:03.755Z",
				"name": "users",
				"type": "auth",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "users_name",
						"name": "firstName",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "ajrktdll",
						"name": "lastName",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "qtf45wv4",
						"name": "copyrightTag",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "vumq2mnp",
						"name": "active",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "users_avatar",
						"name": "avatar",
						"type": "file",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"mimeTypes": [
								"image/jpeg",
								"image/png",
								"image/svg+xml",
								"image/gif",
								"image/webp"
							],
							"thumbs": null,
							"maxSelect": 1,
							"maxSize": 5242880,
							"protected": false
						}
					},
					{
						"system": false,
						"id": "ziheucvm",
						"name": "role",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "fbvy1v7txj0ooy4",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "uspx5aop",
						"name": "projectAssignments",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "bnggeaxuv84cfwh",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": null,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "2cqx32cr",
						"name": "activeProject",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "whgae0tyjp10p6e",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_EmJURt8` + "`" + ` ON ` + "`" + `users` + "`" + ` (\n  ` + "`" + `firstName` + "`" + `,\n  ` + "`" + `lastName` + "`" + `\n)"
				],
				"listRule": "id = @request.auth.id",
				"viewRule": "id = @request.auth.id",
				"createRule": "",
				"updateRule": "id = @request.auth.id",
				"deleteRule": "id = @request.auth.id",
				"options": {
					"allowEmailAuth": true,
					"allowOAuth2Auth": false,
					"allowUsernameAuth": false,
					"exceptEmailDomains": null,
					"manageRule": null,
					"minPasswordLength": 8,
					"onlyEmailDomains": null,
					"onlyVerified": true,
					"requireEmail": false
				}
			},
			{
				"id": "whgae0tyjp10p6e",
				"created": "2024-02-17 12:09:29.586Z",
				"updated": "2024-02-25 17:37:03.599Z",
				"name": "projects",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "ftijuyc3",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "qjxq8787",
						"name": "description",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "ythzcwb6",
						"name": "copyright",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "egzhd45h",
						"name": "copyrightReference",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "xcaerank",
						"name": "locationName",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "gmc6oolx",
						"name": "locationCode",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "0euotsvc",
						"name": "locationCity",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_c7vGiGy` + "`" + ` ON ` + "`" + `projects` + "`" + ` (` + "`" + `name` + "`" + `)",
					"CREATE UNIQUE INDEX ` + "`" + `idx_Exllboh` + "`" + ` ON ` + "`" + `projects` + "`" + ` (` + "`" + `description` + "`" + `)"
				],
				"listRule": "@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.project.id ?= id",
				"viewRule": "@request.auth.role.key = \"admin\" || @request.auth.projectAssignments.project.id ?= id",
				"createRule": "@request.auth.role.key = \"admin\"",
				"updateRule": "@request.auth.role.key = \"admin\"",
				"deleteRule": "@request.auth.role.key = \"admin\"",
				"options": {}
			},
			{
				"id": "5nhk5rl7djdx4lf",
				"created": "2024-02-17 12:22:25.473Z",
				"updated": "2024-02-17 13:50:36.062Z",
				"name": "cameras",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "yvnbmoig",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 3,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "uw8lv2qg",
						"name": "user",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "_pb_users_auth_",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [],
				"listRule": "@request.auth.role.key = \"admin\" || user.id = @request.auth.id",
				"viewRule": "@request.auth.role.key = \"admin\" || user.id = @request.auth.id",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.role.key = \"admin\" || user.id = @request.auth.id",
				"deleteRule": "@request.auth.role.key = \"admin\" || user.id = @request.auth.id",
				"options": {}
			},
			{
				"id": "55ajrfhmhgm37tz",
				"created": "2024-02-17 12:26:13.301Z",
				"updated": "2024-02-17 13:50:59.429Z",
				"name": "batches",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "rdquivab",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "7kzvzxtn",
						"name": "project",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "whgae0tyjp10p6e",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "sjnsakki",
						"name": "user",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "_pb_users_auth_",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [],
				"listRule": "@request.auth.role.key = \"admin\" || @request.auth.role.key = \"projectAdmin\" || @request.auth.id = user.id",
				"viewRule": "@request.auth.role.key = \"admin\" || @request.auth.role.key = \"projectAdmin\" || @request.auth.id = user.id",
				"createRule": "@request.auth.id = user.id",
				"updateRule": "@request.auth.role.key = \"admin\" || @request.auth.role.key = \"projectAdmin\" || @request.auth.id = user.id",
				"deleteRule": "@request.auth.role.key = \"admin\" || @request.auth.role.key = \"projectAdmin\" || @request.auth.id = user.id",
				"options": {}
			},
			{
				"id": "xmc92cdxvv1ijq4",
				"created": "2024-02-17 12:29:38.018Z",
				"updated": "2024-02-17 13:42:15.186Z",
				"name": "image_tags",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "xj6uk2u4",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "dumfhsie",
						"name": "description",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "7lebtjm9",
						"name": "isAlbum",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "8kjl1ist",
						"name": "type",
						"type": "select",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSelect": 1,
							"values": [
								"default",
								"manual",
								"custom"
							]
						}
					},
					{
						"system": false,
						"id": "0taphobx",
						"name": "project",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "whgae0tyjp10p6e",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [],
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "((@request.auth.role.key = \"admin\" || @request.auth.role.key = \"projectAdmin\") && (type = 'default' || type = 'manual')) || (@request.auth.id != '' && type = 'custom')",
				"updateRule": "((@request.auth.role.key = \"admin\" || @request.auth.role.key = \"projectAdmin\") && (type = 'default' || type = 'manual')) || (@request.auth.id != '' && type = 'custom')",
				"deleteRule": "@request.auth.role.key = \"admin\" || @request.auth.role.key = \"projectAdmin\"",
				"options": {}
			},
			{
				"id": "fbvy1v7txj0ooy4",
				"created": "2024-02-17 12:31:05.652Z",
				"updated": "2024-02-17 12:31:05.652Z",
				"name": "roles",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "qlnvjfc7",
						"name": "key",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": 3,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "qkhpvgl3",
						"name": "description",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_uoiRp0Y` + "`" + ` ON ` + "`" + `roles` + "`" + ` (` + "`" + `key` + "`" + `)"
				],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "bnggeaxuv84cfwh",
				"created": "2024-02-17 12:38:32.208Z",
				"updated": "2024-02-17 13:34:28.063Z",
				"name": "project_assignments",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "nesk1ldf",
						"name": "project",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "whgae0tyjp10p6e",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "znlgmynk",
						"name": "user",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "_pb_users_auth_",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "7i5mk6uj",
						"name": "role",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "fbvy1v7txj0ooy4",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_PWNyUfC` + "`" + ` ON ` + "`" + `project_assignments` + "`" + ` (\n  ` + "`" + `project` + "`" + `,\n  ` + "`" + `user` + "`" + `\n)"
				],
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": "@request.auth.role.key = 'admin'",
				"updateRule": "@request.auth.role.key = 'admin'",
				"deleteRule": "@request.auth.role.key = 'admin'",
				"options": {}
			},
			{
				"id": "lm56zd5xql95a0m",
				"created": "2024-02-17 13:48:15.764Z",
				"updated": "2024-02-17 13:48:15.764Z",
				"name": "image_tag_assignments",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "xxtjhi6u",
						"name": "type",
						"type": "select",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSelect": 1,
							"values": [
								"manual",
								"inferred",
								"default"
							]
						}
					},
					{
						"system": false,
						"id": "oi2yo8jv",
						"name": "imageTag",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "xmc92cdxvv1ijq4",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "5020t772ltvs9da",
				"created": "2024-02-17 13:50:08.713Z",
				"updated": "2024-02-17 13:55:44.357Z",
				"name": "images",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "siw47bpx",
						"name": "fileName",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "hnfrmoic",
						"name": "computedFileName",
						"type": "text",
						"required": false,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "vrgr7pui",
						"name": "exifData",
						"type": "json",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSize": 2000000
						}
					},
					{
						"system": false,
						"id": "d6vxlv3q",
						"name": "capturedAt",
						"type": "date",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "",
							"max": ""
						}
					},
					{
						"system": false,
						"id": "zepaatza",
						"name": "capturedAtCorrected",
						"type": "date",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "",
							"max": ""
						}
					},
					{
						"system": false,
						"id": "zdxnxqpn",
						"name": "inferredAt",
						"type": "date",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "",
							"max": ""
						}
					},
					{
						"system": false,
						"id": "1cmvckob",
						"name": "user",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "_pb_users_auth_",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "n5glff7o",
						"name": "imageTagAssignments",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "lm56zd5xql95a0m",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": null,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "iambvaj2",
						"name": "batch",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "55ajrfhmhgm37tz",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "pfz8cxms",
						"name": "project",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "whgae0tyjp10p6e",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "w6wgtnux",
						"name": "camera",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "5nhk5rl7djdx4lf",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [],
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.id != ''",
				"deleteRule": "@request.auth.id != ''",
				"options": {}
			},
			{
				"id": "8k5kgh4acgwhuyo",
				"created": "2024-02-17 13:55:09.509Z",
				"updated": "2024-02-17 13:57:26.658Z",
				"name": "time_offsets",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "eb2ppmtl",
						"name": "serverTime",
						"type": "date",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "",
							"max": ""
						}
					},
					{
						"system": false,
						"id": "2pkawaoa",
						"name": "cameraTime",
						"type": "date",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "",
							"max": ""
						}
					},
					{
						"system": false,
						"id": "bjyjhqgu",
						"name": "camera",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "5nhk5rl7djdx4lf",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [],
				"listRule": "@request.auth.role.key = \"admin\" || camera.user.id = @request.auth.id",
				"viewRule": "@request.auth.role.key = \"admin\" || camera.user.id = @request.auth.id",
				"createRule": "@request.auth.role.key = \"admin\" || camera.user.id = @request.auth.id",
				"updateRule": "@request.auth.role.key = \"admin\"",
				"deleteRule": "@request.auth.role.key = \"admin\"",
				"options": {}
			}
		]`

		collections := []*models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collections); err != nil {
			return err
		}

		return daos.New(db).ImportCollections(collections, true, nil)
	}, func(db dbx.Builder) error {
		return nil
	})
}
