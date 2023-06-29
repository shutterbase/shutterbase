package authorization

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mxcd/go-config/config"
	"github.com/ory/ladon"
	manager "github.com/ory/ladon/manager/memory"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/ent"
)

type UserContext struct {
	Subject      string
	User         *ent.User
	Role         *ent.Role
	ProjectRoles map[string]string
}

var warden *ladon.Ladon

func Init() error {
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

type AuthCheckOptions struct {
	resource  string
	action    Action
	ownerId   *string
	projectId *string
}

func AuthCheckOption() *AuthCheckOptions {
	return &AuthCheckOptions{}
}

func (options *AuthCheckOptions) Resource(resource string) *AuthCheckOptions {
	CONTEXT_PATH := config.Get().String("API_CONTEXT_PATH")
	options.resource = strings.TrimPrefix(resource, CONTEXT_PATH)
	return options
}

func (options *AuthCheckOptions) Action(action Action) *AuthCheckOptions {
	options.action = action
	return options
}

func (options *AuthCheckOptions) OwnerId(ownerId uuid.UUID) *AuthCheckOptions {
	s := ownerId.String()
	options.ownerId = &s
	return options
}

func (options *AuthCheckOptions) ProjectId(projectId uuid.UUID) *AuthCheckOptions {
	s := projectId.String()
	options.projectId = &s
	return options
}

func IsAllowed(c *gin.Context, options *AuthCheckOptions) (bool, error) {
	userContext := GetUserContextFromGinContext(c)
	ladonContext := ladon.Context{
		"userContext": userContext,
	}
	if options.ownerId != nil {
		ladonContext["ownerId"] = *options.ownerId
	}
	if options.projectId != nil {
		ladonContext["projectId"] = *options.projectId
	}
	err := warden.IsAllowed(&ladon.Request{
		Subject:  userContext.Subject,
		Resource: options.resource,
		Action:   options.action.String(),
		Context:  ladonContext,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func IsAdmin(ginContext *gin.Context) bool {
	userContext := GetUserContextFromGinContext(ginContext)
	return userContext.Role.Key == "admin"
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

type OwnerIdCondition struct{}

func (c *OwnerIdCondition) Fulfills(value interface{}, req *ladon.Request) bool {
	s, ok := value.(string)
	userId := req.Context["userContext"].(*UserContext).User.ID.String()

	return ok && s == userId
}

func (c *OwnerIdCondition) GetName() string {
	return "OwnerIdCondition"
}

type ProjectRoleCondition struct {
	Roles []string `json:"roles"`
}

func (c *ProjectRoleCondition) Fulfills(value interface{}, req *ladon.Request) bool {
	roles := c.Roles

	projectsRegex := regexp.MustCompile(`^\/projects\/(?P<ProjectId>[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})(\/.*)?`)
	resource := req.Resource

	res := projectsRegex.FindStringSubmatch(resource)
	if len(res) < 2 {
		return false
	}

	projectId := res[1]

	userContext := req.Context["userContext"].(*UserContext)
	role, ok := userContext.ProjectRoles[projectId]

	if !ok {
		return false
	}

	for _, r := range roles {
		if r == role {
			return true
		}
	}
	
	return false
}

func (c *ProjectRoleCondition) GetName() string {
	return "ProjectRoleCondition"
}
