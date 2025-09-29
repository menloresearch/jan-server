package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/invite"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"

	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type AuthService struct {
	userService         *user.UserService
	apiKeyService       *apikey.ApiKeyService
	organizationService *organization.OrganizationService
	projectService      *project.ProjectService
	inviteService       *invite.InviteService
}

func NewAuthService(
	userService *user.UserService,
	apiKeyService *apikey.ApiKeyService,
	organizationService *organization.OrganizationService,
	projectService *project.ProjectService,
	inviteService *invite.InviteService,
) *AuthService {
	return &AuthService{
		userService,
		apiKeyService,
		organizationService,
		projectService,
		inviteService,
	}
}

const AccessTokenExpirationDuration = 15 * time.Minute
const RefreshTokenExpirationDuration = 7 * 24 * time.Hour

type UserContextKey string

const (
	UserContextKeyEntity UserContextKey = "UserContextKeyEntity"
	UserContextKeyID     UserContextKey = "UserContextKeyID"
)

func (s *AuthService) InitOrganization(ctx context.Context) error {
	orgEntity, err := s.organizationService.FindOrCreateDefaultOrganization(ctx)
	if err != nil {
		return err
	}
	// set DEFAULT_ORGANIZATION
	organization.UpdateDefaultOrganization(orgEntity)

	emails := environment_variables.EnvironmentVariables.ORGANIZATION_ADMIN_EMAILS
	if len(emails) == 0 {
		return fmt.Errorf("no ORGANIZATION_ADMIN_EMAILS configured")
	}

	for _, rawEmail := range emails {
		email := strings.TrimSpace(rawEmail)
		if email == "" {
			continue
		}

		admin, err := s.userService.FindByEmail(ctx, email)
		if err != nil {
			return err
		}
		if admin == nil {
			admin, err = s.RegisterUser(ctx, &user.User{
				Name:    "Admin",
				Email:   email,
				IsGuest: false,
				Enabled: true,
			})
			if err != nil {
				return err
			}
		}

		err = s.organizationService.AddMember(ctx, &organization.OrganizationMember{
			UserID:         admin.ID,
			OrganizationID: orgEntity.ID,
			Role:           organization.OrganizationMemberRoleOwner,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
func (s *AuthService) RegisterUser(ctx context.Context, user *user.User) (*user.User, error) {
	_, err := s.userService.RegisterUser(ctx, user)
	if err != nil {
		return nil, err
	}
	projEntity, err := s.projectService.CreateProjectWithPublicID(ctx, &project.Project{
		Name:           "Default Project",
		Status:         string(project.ProjectStatusActive),
		OrganizationID: organization.DEFAULT_ORGANIZATION.ID,
	})
	if err != nil {
		return nil, err
	}
	err = s.projectService.AddMember(ctx, &project.ProjectMember{
		ProjectID: projEntity.ID,
		UserID:    user.ID,
		Role:      string(project.ProjectMemberRoleOwner),
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) FindOrRegisterUser(ctx context.Context, user *user.User) (*user.User, error) {
	userEntity, err := s.userService.FindByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}
	if userEntity != nil {
		return userEntity, nil
	}
	return s.RegisterUser(ctx, user)
}

func (s *AuthService) HasOrganizationUser(ctx context.Context, email string, orgID uint) (bool, error) {
	user, err := s.userService.FindByEmail(ctx, email)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}
	member, err := s.organizationService.FindOneMemberByFilter(ctx, organization.OrganizationMemberFilter{
		UserID:         &user.ID,
		OrganizationID: &orgID,
	})
	if err != nil {
		return false, err
	}
	if member != nil {
		return true, nil
	}
	return false, nil
}

func (s *AuthService) JWTAuthMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userId, ok := s.getUserPublicIDFromJWT(reqCtx)
		if !ok {
			return
		}
		SetUserIDToContext(reqCtx, userId)
		reqCtx.Next()
	}
}

// Retrieve the user's public ID from the header.
func (s *AuthService) AppUserAuthMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userId, ok := s.getUserPublicIDFromJWT(reqCtx)
		if ok {
			SetUserIDToContext(reqCtx, userId)
			reqCtx.Next()
			return
		}
		userId, ok = s.getUserIDFromApikey(reqCtx)
		if ok {
			SetUserIDToContext(reqCtx, userId)
			reqCtx.Next()
			return
		}

		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "019947f0-eca1-7474-8ed2-09d6e5389b54",
		})
	}
}

func (s *AuthService) AdminUserAuthMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userId, ok := s.getUserPublicIDFromJWT(reqCtx)
		if ok {
			SetUserIDToContext(reqCtx, userId)
			reqCtx.Next()
			return
		}
		userId, ok = s.getUserIDFromAdminkey(reqCtx)
		if ok {
			SetUserIDToContext(reqCtx, userId)
			reqCtx.Next()
			return
		}

		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "4026757e-d5a4-4cf7-8914-2c96f011084f",
		})
	}
}

// Verify user from public ID
func (s *AuthService) RegisteredUserMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		userPublicId, ok := GetUserIDFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "3296ce86-783b-4c05-9fdb-930d3713024e",
			})
			return
		}
		if userPublicId == "" {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "80e1017d-038a-48c1-9de7-c3cdffdddb95",
			})
			return
		}
		user, err := s.userService.FindByPublicID(ctx, userPublicId)
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "6272df83-f538-421b-93ba-c2b6f6d39f39",
			})
			return
		}
		if user == nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "b1ef40e7-9db9-477d-bb59-f3783585195d",
			})
			return
		}
		SetUserToContext(reqCtx, user)
		reqCtx.Next()
	}
}

var OrganizationMemberRuleOwnerOnly = map[string]bool{
	string(organization.OrganizationMemberRoleOwner): true,
}

var OrganizationMemberRuleAll = map[string]bool{
	string(organization.OrganizationMemberRoleOwner):  true,
	string(organization.OrganizationMemberRoleReader): true,
}

func (s *AuthService) getDefaultOrganizationMember(reqCtx *gin.Context) (*organization.OrganizationMember, bool) {
	ctx := reqCtx.Request.Context()
	member, ok := GetAdminOrganizationMemberFromContext(reqCtx)
	if ok {
		return member, ok
	}
	user, ok := GetUserFromContext(reqCtx)
	if !ok || user == nil {
		return nil, false
	}
	membership, err := s.organizationService.FindOneMemberByFilter(ctx, organization.OrganizationMemberFilter{
		UserID:         &user.ID,
		OrganizationID: &organization.DEFAULT_ORGANIZATION.ID,
	})
	if err != nil || membership == nil {
		return nil, false
	}
	return membership, true
}

func (s *AuthService) DefaultOrganizationMemberOptionalMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		SetAdminOrganizationToContext(reqCtx, organization.DEFAULT_ORGANIZATION)
		membership, ok := s.getDefaultOrganizationMember(reqCtx)
		if ok {
			SetAdminOrganizationMemberToContext(reqCtx, membership)
		}
		reqCtx.Next()
	}
}

func (s *AuthService) OrganizationMemberRoleMiddleware(rolesAllowed map[string]bool) gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		SetAdminOrganizationToContext(reqCtx, organization.DEFAULT_ORGANIZATION)
		membership, ok := s.getDefaultOrganizationMember(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "983a8764-6888-450d-a5d5-7442c3904637",
			})
			return
		}
		ok, exists := rolesAllowed[string(membership.Role)]
		if !exists || !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "f0167776-febc-4fc0-a7c1-13a3ba1673ce",
			})
			return
		}
		reqCtx.Next()
	}
}

func (s *AuthService) getUserPublicIDFromJWT(reqCtx *gin.Context) (string, bool) {
	tokenString, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		return "", false
	}
	token, err := jwt.ParseWithClaims(tokenString, &UserClaim{}, func(token *jwt.Token) (interface{}, error) {
		return environment_variables.EnvironmentVariables.JWT_SECRET, nil
	})
	if err != nil || !token.Valid {
		return "", false
	}
	claims, ok := token.Claims.(*UserClaim)
	if !ok {
		return "", false
	}
	return claims.ID, true
}

func (s *AuthService) getUserIDFromApikey(reqCtx *gin.Context) (string, bool) {
	tokenString, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		return "", false
	}
	if !strings.HasPrefix(tokenString, apikey.ApikeyPrefix) {
		return "", false
	}
	token, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		return "", false
	}
	ctx := reqCtx.Request.Context()
	hashed := s.apiKeyService.HashKey(reqCtx, token)
	apikeyEntity, err := s.apiKeyService.FindByKeyHash(ctx, hashed)
	if err != nil {
		return "", false
	}
	if apikeyEntity == nil || apikeyEntity.ApikeyType == string(apikey.ApikeyTypeAdmin) {
		return "", false
	}
	return apikeyEntity.OwnerPublicID, true
}

func (s *AuthService) getUserIDFromAdminkey(reqCtx *gin.Context) (string, bool) {
	tokenString, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		return "", false
	}
	if !strings.HasPrefix(tokenString, apikey.ApikeyPrefix) {
		return "", false
	}
	ctx := reqCtx.Request.Context()
	hashed := s.apiKeyService.HashKey(reqCtx, tokenString)
	apikeyEntity, err := s.apiKeyService.FindByKeyHash(ctx, hashed)
	if err != nil {
		return "", false
	}
	if apikeyEntity == nil || apikeyEntity.ApikeyType != string(apikey.ApikeyTypeAdmin) {
		return "", false
	}

	return apikeyEntity.OwnerPublicID, true
}

func GetUserFromContext(reqCtx *gin.Context) (*user.User, bool) {
	v, ok := reqCtx.Get(string(UserContextKeyEntity))
	if !ok {
		return nil, false
	}
	return v.(*user.User), true
}

func SetUserToContext(reqCtx *gin.Context, user *user.User) {
	reqCtx.Set(string(UserContextKeyEntity), user)
}

func GetUserIDFromContext(reqCtx *gin.Context) (string, bool) {
	userId, ok := reqCtx.Get(string(UserContextKeyID))
	if !ok {
		return "", false
	}
	v, ok := userId.(string)
	if !ok {
		return "", false
	}
	return v, true
}

func SetUserIDToContext(reqCtx *gin.Context, v string) {
	reqCtx.Set(string(UserContextKeyID), v)
}

type ApikeyContextKey string

const (
	ApikeyContextKeyEntity   ApikeyContextKey = "ApikeyContextKeyEntity"
	ApikeyContextKeyPublicID ApikeyContextKey = "apikey_public_id"
)

func (s *AuthService) GetAdminApiKeyFromQuery() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		user, ok := GetUserFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "72ca928d-bd8b-44f8-af70-1a9e33b58295",
			})
			return
		}

		publicID := reqCtx.Param(string(ApikeyContextKeyPublicID))
		if publicID == "" {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "9c6ed28c-1dab-4fab-945a-f0efa2dec1eb",
				Error: "missing apikey public ID",
			})
			return
		}
		adminKeyEntity, err := s.apiKeyService.FindOneByFilter(ctx, apikey.ApiKeyFilter{
			PublicID: &publicID,
		})

		if adminKeyEntity == nil || err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code: "f4f47443-0c80-4c7a-bedc-ac30ec49f494",
			})
			return
		}

		memberEntity, err := s.organizationService.FindOneMemberByFilter(ctx, organization.OrganizationMemberFilter{
			UserID:         &user.ID,
			OrganizationID: adminKeyEntity.OrganizationID,
		})

		if memberEntity == nil || err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "56a9fa87-ddd7-40b7-b2d6-94ae41a600f8",
			})
			return
		}
		SetAdminKeyToContext(reqCtx, adminKeyEntity)
	}
}

func GetAdminKeyFromContext(reqCtx *gin.Context) (*apikey.ApiKey, bool) {
	apiKey, ok := reqCtx.Get(string(ApikeyContextKeyEntity))
	if !ok {
		return nil, false
	}
	v, ok := apiKey.(*apikey.ApiKey)
	if !ok {
		return nil, false
	}
	return v, true
}

func SetAdminKeyToContext(reqCtx *gin.Context, apiKey *apikey.ApiKey) {
	reqCtx.Set(string(ApikeyContextKeyEntity), apiKey)
}

type OrganizationContextKey string

const (
	OrganizationContextKeyEntity       ApikeyContextKey = "OrganizationContextKeyEntity"
	OrganizationContextKeyPublicID     ApikeyContextKey = "org_public_id"
	OrganizationContextKeyMemberEntity ApikeyContextKey = "OrganizationContextKeyMemberEntity"
)

func GetAdminOrganizationFromContext(reqCtx *gin.Context) (*organization.Organization, bool) {
	org, ok := reqCtx.Get(string(OrganizationContextKeyEntity))
	if !ok {
		return nil, false
	}
	v, ok := org.(*organization.Organization)
	if !ok {
		return nil, false
	}
	return v, true
}

func SetAdminOrganizationToContext(reqCtx *gin.Context, org *organization.Organization) {
	reqCtx.Set(string(OrganizationContextKeyEntity), org)
}

func GetAdminOrganizationMemberFromContext(reqCtx *gin.Context) (*organization.OrganizationMember, bool) {
	org, ok := reqCtx.Get(string(OrganizationContextKeyMemberEntity))
	if !ok {
		return nil, false
	}
	v, ok := org.(*organization.OrganizationMember)
	if !ok {
		return nil, false
	}
	return v, true
}

func SetAdminOrganizationMemberToContext(reqCtx *gin.Context, org *organization.OrganizationMember) {
	reqCtx.Set(string(OrganizationContextKeyMemberEntity), org)
}

type ProjectContextKey string

const (
	ProjectContextKeyPublicID ProjectContextKey = "proj_public_id"
	ProjectContextKeyEntity   ProjectContextKey = "ProjectContextKeyEntity"
)

func GetProjectFromContext(reqCtx *gin.Context) (*project.Project, bool) {
	proj, ok := reqCtx.Get(string(ProjectContextKeyEntity))
	if !ok {
		return nil, false
	}
	return proj.(*project.Project), true
}

func SetProjectToContext(reqCtx *gin.Context, project *project.Project) {
	reqCtx.Set(string(ProjectContextKeyEntity), project)
}

func (s *AuthService) AdminProjectMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		user, ok := GetUserFromContext(reqCtx)
		if !ok {
			return
		}
		orgEntity, ok := GetAdminOrganizationFromContext(reqCtx)
		if !ok {
			return
		}
		publicID := reqCtx.Param(string(ProjectContextKeyPublicID))
		if publicID == "" {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "5cbdb58e-6228-4d9a-9893-7f744608a9e8",
				Error: "missing project public ID",
			})
			return
		}
		projectFilter := project.ProjectFilter{
			PublicID:       &publicID,
			OrganizationID: &orgEntity.ID,
		}
		_, ok = GetAdminOrganizationMemberFromContext(reqCtx)
		if !ok {
			projectFilter.MemberID = &user.ID
		}
		proj, err := s.projectService.FindOne(ctx, projectFilter)
		if err != nil || proj == nil {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "121ef112-cb39-4235-9500-b116adb69984",
				Error: "proj not found",
			})
			return
		}
		SetProjectToContext(reqCtx, proj)
		reqCtx.Next()
	}
}

type InviteContextKey string

const (
	InviteContextKeyPublicID InviteContextKey = "invite_public_id"
	InviteContextKeyEntity   InviteContextKey = "InviteContextKeyEntity"
)

func GetAdminInviteFromContext(reqCtx *gin.Context) (*invite.Invite, bool) {
	i, ok := reqCtx.Get(string(InviteContextKeyEntity))
	if !ok {
		return nil, false
	}
	v, ok := i.(*invite.Invite)
	if !ok {
		return nil, false
	}
	return v, true
}

func SetAdminInviteToContext(reqCtx *gin.Context, i *invite.Invite) {
	reqCtx.Set(string(InviteContextKeyEntity), i)
}

func (s *AuthService) AdminInviteMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		orgEntity, ok := GetAdminOrganizationFromContext(reqCtx)
		if !ok {
			return
		}
		publicID := reqCtx.Param(string(InviteContextKeyPublicID))
		if publicID == "" {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "5cbdb58e-6228-4d9a-9893-7f744608a9e8",
				Error: "missing invite public ID",
			})
			return
		}

		inviteEntity, err := s.inviteService.FindOne(ctx, invite.InvitesFilter{
			PublicID:       &publicID,
			OrganizationID: &orgEntity.ID,
		})
		if err != nil || inviteEntity == nil {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "2daa8be0-df7d-4faa-ba4d-00c4dae8ceae",
				Error: "invite not found",
			})
			return
		}
		SetAdminInviteToContext(reqCtx, inviteEntity)
		reqCtx.Next()
	}
}
