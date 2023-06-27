package authorization

import (
	"github.com/gin-gonic/gin"
	"github.com/ory/ladon"
	manager "github.com/ory/ladon/manager/memory"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/ent"
)

var policies = []*ladon.DefaultPolicy{
	{
		ID:          "7d708b20-8858-4e31-8cc3-752ebe11c139",
		Description: "Allow anonymous access to health endpoint",
		Subjects:    []string{"anonymous"},
		Resources:   []string{"/health"},
		Actions:     R.GetItems(),
		Conditions:  ladon.Conditions{},
		Effect:      ladon.AllowAccess,
	},
	{
		ID:          "b7c92c8a-38dc-4f0d-9f19-cf9e0bd93f73",
		Description: "Allow unauthenticated request access",
		Subjects:    []string{"anonymous"},
		Resources:   []string{"/health", "/register", "/confirm", "/login", "/logout", "/refresh", "request-password-reset", "/password-reset"},
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
	{
		ID:          "ffcf103a-99eb-4cda-ba85-4de52b772b2a",
		Description: "Allow request handling for all authenticated users",
		Subjects:    []string{"role:user"},
		Resources:   []string{"/users", "/users/<.+>"},
		Actions:     []string{REQUEST.String()},
		Conditions:  ladon.Conditions{},
		Effect:      ladon.AllowAccess,
	},
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
		Resources:   []string{"/users/me", "/users/:id"},
		Actions:     RUD.GetItems(),
		Conditions: ladon.Conditions{
			"resourceId": &OwnUserIdCondition{},
		},
		Effect: ladon.AllowAccess,
	},
}

type UserContext struct {
	Subject      string
	User         *ent.User
	Role         *ent.Role
	ProjectRoles map[string]string
}

var warden *ladon.Ladon

func Init() error {
	// ladon.ConditionFactories[new(OwnUserIdCondition).GetName()] = func() OwnUserIdCondition {
	// 	return new(OwnUserIdCondition)
	// }
	warden = &ladon.Ladon{
		Manager: manager.NewMemoryManager(),
	}
	for _, policy := range policies {
		err := warden.Manager.Create(policy)
		if err != nil {
			return err
		}
	}
	return nil
}

func IsAllowed(ginContext *gin.Context, resource string, action Action, resourceId string) (bool, error) {
	userContext := GetUserContextFromGinContext(ginContext)
	ladonContext := ladon.Context{
		"userContext": userContext,
		"resourceId":  resourceId,
	}
	err := warden.IsAllowed(&ladon.Request{
		Subject:  userContext.Subject,
		Resource: resource,
		Action:   action.String(),
		Context:  ladonContext,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

type ActionCombination string

const (
	C    ActionCombination = "C"
	R    ActionCombination = "R"
	U    ActionCombination = "U"
	D    ActionCombination = "D"
	CR   ActionCombination = "CR"
	CU   ActionCombination = "CU"
	CD   ActionCombination = "CD"
	RU   ActionCombination = "RU"
	RD   ActionCombination = "RD"
	UD   ActionCombination = "UD"
	CRU  ActionCombination = "CRU"
	CRD  ActionCombination = "CRD"
	CUD  ActionCombination = "CUD"
	RUD  ActionCombination = "RUD"
	CRUD ActionCombination = "CRUD"
)

type Action string

const (
	CREATE  Action = "CREATE"
	READ    Action = "READ"
	UPDATE  Action = "UPDATE"
	DELETE  Action = "DELETE"
	REQUEST Action = "REQUEST"
)

func (action Action) String() string {
	return string(action)
}

func (action ActionCombination) GetItems() []string {
	switch action {
	case C:
		return []string{"CREATE"}
	case R:
		return []string{"READ"}
	case U:
		return []string{"UPDATE"}
	case D:
		return []string{"DELETE"}
	case CR:
		return []string{"CREATE", "READ"}
	case CU:
		return []string{"CREATE", "UPDATE"}
	case CD:
		return []string{"CREATE", "DELETE"}
	case RU:
		return []string{"READ", "UPDATE"}
	case RD:
		return []string{"READ", "DELETE"}
	case UD:
		return []string{"UPDATE", "DELETE"}
	case CRU:
		return []string{"CREATE", "READ", "UPDATE"}
	case CRD:
		return []string{"CREATE", "READ", "DELETE"}
	case CUD:
		return []string{"CREATE", "UPDATE", "DELETE"}
	case RUD:
		return []string{"READ", "UPDATE", "DELETE"}
	case CRUD:
		return []string{"CREATE", "READ", "UPDATE", "DELETE"}
	default:
		return []string{}
	}
}

func GetUserContextFromGinContext(c *gin.Context) *UserContext {
	contextValue, ok := c.Get("userContext")
	if !ok {
		log.Panic().Msg("userContext not set in gin context")
	}
	return contextValue.(*UserContext)
}

type OwnUserIdCondition struct {
	Id string `json:"id"`
}

func (c *OwnUserIdCondition) Fulfills(value interface{}, req *ladon.Request) bool {
	s, ok := value.(string)
	userId := req.Context["userContext"].(*UserContext).User.ID.String()

	return ok && s == userId
}

func (c *OwnUserIdCondition) GetName() string {
	return "OwnUserIdCondition"
}
