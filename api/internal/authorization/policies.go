package authorization

import "github.com/ory/ladon"

const UUID_REGEX string = "<[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}>"

var policies = []*ladon.DefaultPolicy{
	{
		ID:          "7d708b20-8858-4e31-8cc3-752ebe11c139",
		Description: "Allow anonymous access to health endpoint",
		Subjects:    []string{"<.+>"},
		Resources:   []string{"/health"},
		Actions:     []string{REQUEST.String(), READ.String()},
		Conditions:  ladon.Conditions{},
		Effect:      ladon.AllowAccess,
	},
	{
		ID:          "b7c92c8a-38dc-4f0d-9f19-cf9e0bd93f73",
		Description: "Allow unauthenticated request access",
		Subjects:    []string{"anonymous"},
		Resources:   []string{"/register", "/confirm", "/login", "/logout", "/refresh", "request-password-reset", "/password-reset"},
		Actions:     []string{REQUEST.String()},
		Conditions:  ladon.Conditions{},
		Effect:      ladon.AllowAccess,
	},
	{
		ID:          "3513b134-b3d3-42b5-bfde-7299ea3c1c8a",
		Description: "Allow authenticated request access",
		Subjects:    []string{"role:user", "role:admin"},
		Resources:   []string{"/<.+>"},
		Actions:     []string{REQUEST.String()},
		Conditions:  ladon.Conditions{},
		Effect:      ladon.AllowAccess,
	},
	// {
	// 	ID:          "ffcf103a-99eb-4cda-ba85-4de52b772b2a",
	// 	Description: "Allow request handling for all authenticated users",
	// 	Subjects:    []string{"role:user"},
	// 	Resources:   []string{"/users", "/users/" + UUID_REGEX},
	// 	Actions:     []string{REQUEST.String()},
	// 	Conditions:  ladon.Conditions{},
	// 	Effect:      ladon.AllowAccess,
	// },
	{
		ID:          "adfdb95b-ccac-4690-8321-bb064d6c8160",
		Description: "Allow all Action on admin user",
		Subjects:    []string{"role:admin"},
		Resources:   []string{"/<.+>"},
		Actions:     []string{"<.+>"},
		Conditions:  ladon.Conditions{},
		Effect:      ladon.AllowAccess,
	},
	{
		ID:          "cba4a5fc-cb90-4109-9d4c-7518abaea57e",
		Description: "Allow own user read access",
		Subjects:    []string{"role:user"},
		Resources:   []string{"/users/me", "/users/" + UUID_REGEX},
		Actions:     RUD.GetItems(),
		Conditions: ladon.Conditions{
			"ownerId": &OwnerIdCondition{},
		},
		Effect: ladon.AllowAccess,
	},
	{
		ID:          "77db29de-b300-41d7-8950-8b30001bc925",
		Description: "Allow minimal users list read access",
		Subjects:    []string{"role:user"},
		Resources:   []string{"/users/minimal"},
		Actions:     R.GetItems(),
		Conditions:  ladon.Conditions{},
		Effect:      ladon.AllowAccess,
	},
	{
		ID:          "e12638b7-3fab-4991-aafc-c6917a208d3e",
		Description: "Allow roles access for user",
		Subjects:    []string{"role:user"},
		Resources:   []string{"/roles", "/roles/" + UUID_REGEX},
		Actions:     R.GetItems(),
		Conditions:  ladon.Conditions{},
		Effect:      ladon.AllowAccess,
	},
	{
		ID:          "a54e0d76-98d3-459f-a144-66e07ee2410f",
		Description: "Allow project access and creation for user",
		Subjects:    []string{"role:user"},
		Resources:   []string{"/projects", "/projects/" + UUID_REGEX},
		Actions:     CR.GetItems(),
		Conditions:  ladon.Conditions{},
		Effect:      ladon.AllowAccess,
	},
	{
		ID:          "6411e2c9-e58f-4420-9536-b6bd4497bc62",
		Description: "Allow actions for project admin",
		Subjects:    []string{"role:user"},
		Resources:   []string{"/projects", "/projects/" + UUID_REGEX},
		Actions:     CRUD.GetItems(),
		Conditions: ladon.Conditions{
			"project": &ProjectRoleCondition{
				Roles: []string{"role:project_admin"},
			},
		},
		Effect: ladon.AllowAccess,
	},
	{
		ID:          "45a20b6b-f30e-47d6-95ad-8b17ef272f9a",
		Description: "Allow user access to its own cameras",
		Subjects:    []string{"role:user"},
		Resources:   []string{"/users/" + UUID_REGEX + "/cameras", "/users/" + UUID_REGEX + "/cameras/" + UUID_REGEX},
		Actions:     CRUD.GetItems(),
		Conditions: ladon.Conditions{
			"userId": &OwnUserPathCondition{},
		},
		Effect: ladon.AllowAccess,
	},
}
