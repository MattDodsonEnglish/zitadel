package query

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/lib/pq"

	"github.com/caos/zitadel/internal/domain"
	errs "github.com/caos/zitadel/internal/errors"
)

var (
	userGrantStmt = regexp.QuoteMeta(
		"SELECT zitadel.projections.user_grants.id" +
			", zitadel.projections.user_grants.creation_date" +
			", zitadel.projections.user_grants.change_date" +
			", zitadel.projections.user_grants.sequence" +
			", zitadel.projections.user_grants.grant_id" +
			", zitadel.projections.user_grants.roles" +
			", zitadel.projections.user_grants.state" +
			", zitadel.projections.user_grants.user_id" +
			", zitadel.projections.users.username" +
			", zitadel.projections.users.type" +
			", zitadel.projections.users.resource_owner" +
			", zitadel.projections.users_humans.first_name" +
			", zitadel.projections.users_humans.last_name" +
			", zitadel.projections.users_humans.email" +
			", zitadel.projections.users_humans.display_name" +
			", zitadel.projections.users_humans.avatar_key" +
			", zitadel.projections.login_names.login_name" +
			", zitadel.projections.user_grants.resource_owner" +
			", zitadel.projections.orgs.name" +
			", zitadel.projections.orgs.primary_domain" +
			", zitadel.projections.user_grants.project_id" +
			", zitadel.projections.projects.name" +
			" FROM zitadel.projections.user_grants" +
			" LEFT JOIN zitadel.projections.users ON zitadel.projections.user_grants.user_id = zitadel.projections.users.id" +
			" LEFT JOIN zitadel.projections.users_humans ON zitadel.projections.user_grants.user_id = zitadel.projections.users_humans.user_id" +
			" LEFT JOIN zitadel.projections.orgs ON zitadel.projections.user_grants.resource_owner = zitadel.projections.orgs.id" +
			" LEFT JOIN zitadel.projections.projects ON zitadel.projections.user_grants.project_id = zitadel.projections.projects.id" +
			" LEFT JOIN zitadel.projections.login_names ON zitadel.projections.user_grants.user_id = zitadel.projections.login_names.user_id" +
			" WHERE zitadel.projections.login_names.is_primary = $1")
	userGrantCols = []string{
		"id",
		"creation_date",
		"change_date",
		"sequence",
		"grant_id",
		"roles",
		"state",
		"user_id",
		"username",
		"type",
		"resource_owner", //user resource owner
		"first_name",
		"last_name",
		"email",
		"display_name",
		"avatar_key",
		"login_name",
		"resource_owner", //user_grant resource owner
		"name",           //org name
		"primary_domain",
		"project_id",
		"name", //project name
	}
	userGrantsStmt = regexp.QuoteMeta(
		"SELECT zitadel.projections.user_grants.id" +
			", zitadel.projections.user_grants.creation_date" +
			", zitadel.projections.user_grants.change_date" +
			", zitadel.projections.user_grants.sequence" +
			", zitadel.projections.user_grants.grant_id" +
			", zitadel.projections.user_grants.roles" +
			", zitadel.projections.user_grants.state" +
			", zitadel.projections.user_grants.user_id" +
			", zitadel.projections.users.username" +
			", zitadel.projections.users.type" +
			", zitadel.projections.users.resource_owner" +
			", zitadel.projections.users_humans.first_name" +
			", zitadel.projections.users_humans.last_name" +
			", zitadel.projections.users_humans.email" +
			", zitadel.projections.users_humans.display_name" +
			", zitadel.projections.users_humans.avatar_key" +
			", zitadel.projections.login_names.login_name" +
			", zitadel.projections.user_grants.resource_owner" +
			", zitadel.projections.orgs.name" +
			", zitadel.projections.orgs.primary_domain" +
			", zitadel.projections.user_grants.project_id" +
			", zitadel.projections.projects.name" +
			", COUNT(*) OVER ()" +
			" FROM zitadel.projections.user_grants" +
			" LEFT JOIN zitadel.projections.users ON zitadel.projections.user_grants.user_id = zitadel.projections.users.id" +
			" LEFT JOIN zitadel.projections.users_humans ON zitadel.projections.user_grants.user_id = zitadel.projections.users_humans.user_id" +
			" LEFT JOIN zitadel.projections.orgs ON zitadel.projections.user_grants.resource_owner = zitadel.projections.orgs.id" +
			" LEFT JOIN zitadel.projections.projects ON zitadel.projections.user_grants.project_id = zitadel.projections.projects.id" +
			" LEFT JOIN zitadel.projections.login_names ON zitadel.projections.user_grants.user_id = zitadel.projections.login_names.user_id" +
			" WHERE zitadel.projections.login_names.is_primary = $1")
	userGrantsCols = append(
		userGrantCols,
		"count",
	)
)

func Test_UserGrantPrepares(t *testing.T) {
	type want struct {
		sqlExpectations sqlExpectation
		err             checkErr
	}
	tests := []struct {
		name    string
		prepare interface{}
		want    want
		object  interface{}
	}{
		{
			name:    "prepareUserGrantQuery no result",
			prepare: prepareUserGrantQuery,
			want: want{
				sqlExpectations: mockQueries(
					userGrantStmt,
					nil,
					nil,
				),
				err: func(err error) (error, bool) {
					if !errs.IsNotFound(err) {
						return fmt.Errorf("err should be zitadel.NotFoundError got: %w", err), false
					}
					return nil, true
				},
			},
			object: (*UserGrant)(nil),
		},
		{
			name:    "prepareUserGrantQuery found",
			prepare: prepareUserGrantQuery,
			want: want{
				sqlExpectations: mockQuery(
					userGrantStmt,
					userGrantCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						20211111,
						"grant-id",
						pq.StringArray{"role-key"},
						domain.UserGrantStateActive,
						"user-id",
						"username",
						domain.UserTypeHuman,
						"resource-owner",
						"first-name",
						"last-name",
						"email",
						"display-name",
						"avatar-key",
						"login-name",
						"ro",
						"org-name",
						"primary-domain",
						"project-id",
						"project-name",
					},
				),
			},
			object: &UserGrant{
				ID:                 "id",
				CreationDate:       testNow,
				ChangeDate:         testNow,
				Sequence:           20211111,
				Roles:              []string{"role-key"},
				GrantID:            "grant-id",
				State:              domain.UserGrantStateActive,
				UserID:             "user-id",
				Username:           "username",
				UserType:           domain.UserTypeHuman,
				UserResourceOwner:  "resource-owner",
				FirstName:          "first-name",
				LastName:           "last-name",
				Email:              "email",
				DisplayName:        "display-name",
				AvatarURL:          "avatar-key",
				PreferredLoginName: "login-name",
				ResourceOwner:      "ro",
				OrgName:            "org-name",
				OrgPrimaryDomain:   "primary-domain",
				ProjectID:          "project-id",
				ProjectName:        "project-name",
			},
		},
		{
			name:    "prepareUserGrantQuery machine user found",
			prepare: prepareUserGrantQuery,
			want: want{
				sqlExpectations: mockQuery(
					userGrantStmt,
					userGrantCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						20211111,
						"grant-id",
						pq.StringArray{"role-key"},
						domain.UserGrantStateActive,
						"user-id",
						"username",
						domain.UserTypeMachine,
						"resource-owner",
						nil,
						nil,
						nil,
						nil,
						nil,
						"login-name",
						"ro",
						"org-name",
						"primary-domain",
						"project-id",
						"project-name",
					},
				),
			},
			object: &UserGrant{
				ID:                 "id",
				CreationDate:       testNow,
				ChangeDate:         testNow,
				Sequence:           20211111,
				Roles:              []string{"role-key"},
				GrantID:            "grant-id",
				State:              domain.UserGrantStateActive,
				UserID:             "user-id",
				Username:           "username",
				UserType:           domain.UserTypeMachine,
				UserResourceOwner:  "resource-owner",
				FirstName:          "",
				LastName:           "",
				Email:              "",
				DisplayName:        "",
				AvatarURL:          "",
				PreferredLoginName: "login-name",
				ResourceOwner:      "ro",
				OrgName:            "org-name",
				OrgPrimaryDomain:   "primary-domain",
				ProjectID:          "project-id",
				ProjectName:        "project-name",
			},
		},
		{
			name:    "prepareUserGrantQuery (no org) found",
			prepare: prepareUserGrantQuery,
			want: want{
				sqlExpectations: mockQuery(
					userGrantStmt,
					userGrantCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						20211111,
						"grant-id",
						pq.StringArray{"role-key"},
						domain.UserGrantStateActive,
						"user-id",
						"username",
						domain.UserTypeHuman,
						"resource-owner",
						"first-name",
						"last-name",
						"email",
						"display-name",
						"avatar-key",
						"login-name",
						"ro",
						nil,
						nil,
						"project-id",
						"project-name",
					},
				),
			},
			object: &UserGrant{
				ID:                 "id",
				CreationDate:       testNow,
				ChangeDate:         testNow,
				Sequence:           20211111,
				Roles:              []string{"role-key"},
				GrantID:            "grant-id",
				State:              domain.UserGrantStateActive,
				UserID:             "user-id",
				Username:           "username",
				UserType:           domain.UserTypeHuman,
				UserResourceOwner:  "resource-owner",
				FirstName:          "first-name",
				LastName:           "last-name",
				Email:              "email",
				DisplayName:        "display-name",
				AvatarURL:          "avatar-key",
				PreferredLoginName: "login-name",
				ResourceOwner:      "ro",
				OrgName:            "",
				OrgPrimaryDomain:   "",
				ProjectID:          "project-id",
				ProjectName:        "project-name",
			},
		},
		{
			name:    "prepareUserGrantQuery (no project) found",
			prepare: prepareUserGrantQuery,
			want: want{
				sqlExpectations: mockQuery(
					userGrantStmt,
					userGrantCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						20211111,
						"grant-id",
						pq.StringArray{"role-key"},
						domain.UserGrantStateActive,
						"user-id",
						"username",
						domain.UserTypeHuman,
						"resource-owner",
						"first-name",
						"last-name",
						"email",
						"display-name",
						"avatar-key",
						"login-name",
						"ro",
						"org-name",
						"primary-domain",
						"project-id",
						nil,
					},
				),
			},
			object: &UserGrant{
				ID:                 "id",
				CreationDate:       testNow,
				ChangeDate:         testNow,
				Sequence:           20211111,
				Roles:              []string{"role-key"},
				GrantID:            "grant-id",
				State:              domain.UserGrantStateActive,
				UserID:             "user-id",
				Username:           "username",
				UserType:           domain.UserTypeHuman,
				UserResourceOwner:  "resource-owner",
				FirstName:          "first-name",
				LastName:           "last-name",
				Email:              "email",
				DisplayName:        "display-name",
				AvatarURL:          "avatar-key",
				PreferredLoginName: "login-name",
				ResourceOwner:      "ro",
				OrgName:            "org-name",
				OrgPrimaryDomain:   "primary-domain",
				ProjectID:          "project-id",
				ProjectName:        "",
			},
		},
		{
			name:    "prepareUserGrantQuery (no loginname) found",
			prepare: prepareUserGrantQuery,
			want: want{
				sqlExpectations: mockQuery(
					userGrantStmt,
					userGrantCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						20211111,
						"grant-id",
						pq.StringArray{"role-key"},
						domain.UserGrantStateActive,
						"user-id",
						"username",
						domain.UserTypeHuman,
						"resource-owner",
						"first-name",
						"last-name",
						"email",
						"display-name",
						"avatar-key",
						nil,
						"ro",
						"org-name",
						"primary-domain",
						"project-id",
						"project-name",
					},
				),
			},
			object: &UserGrant{
				ID:                 "id",
				CreationDate:       testNow,
				ChangeDate:         testNow,
				Sequence:           20211111,
				Roles:              []string{"role-key"},
				GrantID:            "grant-id",
				State:              domain.UserGrantStateActive,
				UserID:             "user-id",
				Username:           "username",
				UserType:           domain.UserTypeHuman,
				UserResourceOwner:  "resource-owner",
				FirstName:          "first-name",
				LastName:           "last-name",
				Email:              "email",
				DisplayName:        "display-name",
				AvatarURL:          "avatar-key",
				PreferredLoginName: "",
				ResourceOwner:      "ro",
				OrgName:            "org-name",
				OrgPrimaryDomain:   "primary-domain",
				ProjectID:          "project-id",
				ProjectName:        "project-name",
			},
		},
		{
			name:    "prepareUserGrantQuery sql err",
			prepare: prepareUserGrantQuery,
			want: want{
				sqlExpectations: mockQueryErr(
					userGrantStmt,
					sql.ErrConnDone,
				),
				err: func(err error) (error, bool) {
					if !errors.Is(err, sql.ErrConnDone) {
						return fmt.Errorf("err should be sql.ErrConnDone got: %w", err), false
					}
					return nil, true
				},
			},
			object: nil,
		},
		{
			name:    "prepareUserGrantsQuery no result",
			prepare: prepareUserGrantsQuery,
			want: want{
				sqlExpectations: mockQueries(
					userGrantsStmt,
					nil,
					nil,
				),
			},
			object: &UserGrants{UserGrants: []*UserGrant{}},
		},
		{
			name:    "prepareUserGrantsQuery one grant",
			prepare: prepareUserGrantsQuery,
			want: want{
				sqlExpectations: mockQueries(
					userGrantsStmt,
					userGrantsCols,
					[][]driver.Value{
						{
							"id",
							testNow,
							testNow,
							20211111,
							"grant-id",
							pq.StringArray{"role-key"},
							domain.UserGrantStateActive,
							"user-id",
							"username",
							domain.UserTypeHuman,
							"resource-owner",
							"first-name",
							"last-name",
							"email",
							"display-name",
							"avatar-key",
							"login-name",
							"ro",
							"org-name",
							"primary-domain",
							"project-id",
							"project-name",
						},
					},
				),
			},
			object: &UserGrants{
				SearchResponse: SearchResponse{
					Count: 1,
				},
				UserGrants: []*UserGrant{
					{
						ID:                 "id",
						CreationDate:       testNow,
						ChangeDate:         testNow,
						Sequence:           20211111,
						Roles:              []string{"role-key"},
						GrantID:            "grant-id",
						State:              domain.UserGrantStateActive,
						UserID:             "user-id",
						Username:           "username",
						UserType:           domain.UserTypeHuman,
						UserResourceOwner:  "resource-owner",
						FirstName:          "first-name",
						LastName:           "last-name",
						Email:              "email",
						DisplayName:        "display-name",
						AvatarURL:          "avatar-key",
						PreferredLoginName: "login-name",
						ResourceOwner:      "ro",
						OrgName:            "org-name",
						OrgPrimaryDomain:   "primary-domain",
						ProjectID:          "project-id",
						ProjectName:        "project-name",
					},
				},
			},
		},
		{
			name:    "prepareUserGrantsQuery one grant (machine user)",
			prepare: prepareUserGrantsQuery,
			want: want{
				sqlExpectations: mockQueries(
					userGrantsStmt,
					userGrantsCols,
					[][]driver.Value{
						{
							"id",
							testNow,
							testNow,
							20211111,
							"grant-id",
							pq.StringArray{"role-key"},
							domain.UserGrantStateActive,
							"user-id",
							"username",
							domain.UserTypeMachine,
							"resource-owner",
							nil,
							nil,
							nil,
							nil,
							nil,
							"login-name",
							"ro",
							"org-name",
							"primary-domain",
							"project-id",
							"project-name",
						},
					},
				),
			},
			object: &UserGrants{
				SearchResponse: SearchResponse{
					Count: 1,
				},
				UserGrants: []*UserGrant{
					{
						ID:                 "id",
						CreationDate:       testNow,
						ChangeDate:         testNow,
						Sequence:           20211111,
						Roles:              []string{"role-key"},
						GrantID:            "grant-id",
						State:              domain.UserGrantStateActive,
						UserID:             "user-id",
						Username:           "username",
						UserType:           domain.UserTypeMachine,
						UserResourceOwner:  "resource-owner",
						FirstName:          "",
						LastName:           "",
						Email:              "",
						DisplayName:        "",
						AvatarURL:          "",
						PreferredLoginName: "login-name",
						ResourceOwner:      "ro",
						OrgName:            "org-name",
						OrgPrimaryDomain:   "primary-domain",
						ProjectID:          "project-id",
						ProjectName:        "project-name",
					},
				},
			},
		},
		{
			name:    "prepareUserGrantsQuery one grant (no org)",
			prepare: prepareUserGrantsQuery,
			want: want{
				sqlExpectations: mockQueries(
					userGrantsStmt,
					userGrantsCols,
					[][]driver.Value{
						{
							"id",
							testNow,
							testNow,
							20211111,
							"grant-id",
							pq.StringArray{"role-key"},
							domain.UserGrantStateActive,
							"user-id",
							"username",
							domain.UserTypeMachine,
							"resource-owner",
							"first-name",
							"last-name",
							"email",
							"display-name",
							"avatar-key",
							"login-name",
							"ro",
							nil,
							nil,
							"project-id",
							"project-name",
						},
					},
				),
			},
			object: &UserGrants{
				SearchResponse: SearchResponse{
					Count: 1,
				},
				UserGrants: []*UserGrant{
					{
						ID:                 "id",
						CreationDate:       testNow,
						ChangeDate:         testNow,
						Sequence:           20211111,
						Roles:              []string{"role-key"},
						GrantID:            "grant-id",
						State:              domain.UserGrantStateActive,
						UserID:             "user-id",
						Username:           "username",
						UserType:           domain.UserTypeMachine,
						UserResourceOwner:  "resource-owner",
						FirstName:          "first-name",
						LastName:           "last-name",
						Email:              "email",
						DisplayName:        "display-name",
						AvatarURL:          "avatar-key",
						PreferredLoginName: "login-name",
						ResourceOwner:      "ro",
						OrgName:            "",
						OrgPrimaryDomain:   "",
						ProjectID:          "project-id",
						ProjectName:        "project-name",
					},
				},
			},
		},
		{
			name:    "prepareUserGrantsQuery one grant (no project)",
			prepare: prepareUserGrantsQuery,
			want: want{
				sqlExpectations: mockQueries(
					userGrantsStmt,
					userGrantsCols,
					[][]driver.Value{
						{
							"id",
							testNow,
							testNow,
							20211111,
							"grant-id",
							pq.StringArray{"role-key"},
							domain.UserGrantStateActive,
							"user-id",
							"username",
							domain.UserTypeHuman,
							"resource-owner",
							"first-name",
							"last-name",
							"email",
							"display-name",
							"avatar-key",
							"login-name",
							"ro",
							"org-name",
							"primary-domain",
							"project-id",
							nil,
						},
					},
				),
			},
			object: &UserGrants{
				SearchResponse: SearchResponse{
					Count: 1,
				},
				UserGrants: []*UserGrant{
					{
						ID:                 "id",
						CreationDate:       testNow,
						ChangeDate:         testNow,
						Sequence:           20211111,
						Roles:              []string{"role-key"},
						GrantID:            "grant-id",
						State:              domain.UserGrantStateActive,
						UserID:             "user-id",
						Username:           "username",
						UserType:           domain.UserTypeHuman,
						UserResourceOwner:  "resource-owner",
						FirstName:          "first-name",
						LastName:           "last-name",
						Email:              "email",
						DisplayName:        "display-name",
						AvatarURL:          "avatar-key",
						PreferredLoginName: "login-name",
						ResourceOwner:      "ro",
						OrgName:            "org-name",
						OrgPrimaryDomain:   "primary-domain",
						ProjectID:          "project-id",
						ProjectName:        "",
					},
				},
			},
		},
		{
			name:    "prepareUserGrantsQuery one grant (no loginname)",
			prepare: prepareUserGrantsQuery,
			want: want{
				sqlExpectations: mockQueries(
					userGrantsStmt,
					userGrantsCols,
					[][]driver.Value{
						{
							"id",
							testNow,
							testNow,
							20211111,
							"grant-id",
							pq.StringArray{"role-key"},
							domain.UserGrantStateActive,
							"user-id",
							"username",
							domain.UserTypeHuman,
							"resource-owner",
							"first-name",
							"last-name",
							"email",
							"display-name",
							"avatar-key",
							nil,
							"ro",
							"org-name",
							"primary-domain",
							"project-id",
							"project-name",
						},
					},
				),
			},
			object: &UserGrants{
				SearchResponse: SearchResponse{
					Count: 1,
				},
				UserGrants: []*UserGrant{
					{
						ID:                 "id",
						CreationDate:       testNow,
						ChangeDate:         testNow,
						Sequence:           20211111,
						Roles:              []string{"role-key"},
						GrantID:            "grant-id",
						State:              domain.UserGrantStateActive,
						UserID:             "user-id",
						Username:           "username",
						UserType:           domain.UserTypeHuman,
						UserResourceOwner:  "resource-owner",
						FirstName:          "first-name",
						LastName:           "last-name",
						Email:              "email",
						DisplayName:        "display-name",
						AvatarURL:          "avatar-key",
						PreferredLoginName: "",
						ResourceOwner:      "ro",
						OrgName:            "org-name",
						OrgPrimaryDomain:   "primary-domain",
						ProjectID:          "project-id",
						ProjectName:        "project-name",
					},
				},
			},
		},
		{
			name:    "prepareUserGrantsQuery multiple grants",
			prepare: prepareUserGrantsQuery,
			want: want{
				sqlExpectations: mockQueries(
					userGrantsStmt,
					userGrantsCols,
					[][]driver.Value{
						{
							"id",
							testNow,
							testNow,
							20211111,
							"grant-id",
							pq.StringArray{"role-key"},
							domain.UserGrantStateActive,
							"user-id",
							"username",
							domain.UserTypeHuman,
							"resource-owner",
							"first-name",
							"last-name",
							"email",
							"display-name",
							"avatar-key",
							"login-name",
							"ro",
							"org-name",
							"primary-domain",
							"project-id",
							"project-name",
						},
						{
							"id",
							testNow,
							testNow,
							20211111,
							"grant-id",
							pq.StringArray{"role-key"},
							domain.UserGrantStateActive,
							"user-id",
							"username",
							domain.UserTypeHuman,
							"resource-owner",
							"first-name",
							"last-name",
							"email",
							"display-name",
							"avatar-key",
							"login-name",
							"ro",
							"org-name",
							"primary-domain",
							"project-id",
							"project-name",
						},
					},
				),
			},
			object: &UserGrants{
				SearchResponse: SearchResponse{
					Count: 2,
				},
				UserGrants: []*UserGrant{
					{
						ID:                 "id",
						CreationDate:       testNow,
						ChangeDate:         testNow,
						Sequence:           20211111,
						Roles:              []string{"role-key"},
						GrantID:            "grant-id",
						State:              domain.UserGrantStateActive,
						UserID:             "user-id",
						Username:           "username",
						UserType:           domain.UserTypeHuman,
						UserResourceOwner:  "resource-owner",
						FirstName:          "first-name",
						LastName:           "last-name",
						Email:              "email",
						DisplayName:        "display-name",
						AvatarURL:          "avatar-key",
						PreferredLoginName: "login-name",
						ResourceOwner:      "ro",
						OrgName:            "org-name",
						OrgPrimaryDomain:   "primary-domain",
						ProjectID:          "project-id",
						ProjectName:        "project-name",
					},
					{
						ID:                 "id",
						CreationDate:       testNow,
						ChangeDate:         testNow,
						Sequence:           20211111,
						Roles:              []string{"role-key"},
						GrantID:            "grant-id",
						State:              domain.UserGrantStateActive,
						UserID:             "user-id",
						Username:           "username",
						UserType:           domain.UserTypeHuman,
						UserResourceOwner:  "resource-owner",
						FirstName:          "first-name",
						LastName:           "last-name",
						Email:              "email",
						DisplayName:        "display-name",
						AvatarURL:          "avatar-key",
						PreferredLoginName: "login-name",
						ResourceOwner:      "ro",
						OrgName:            "org-name",
						OrgPrimaryDomain:   "primary-domain",
						ProjectID:          "project-id",
						ProjectName:        "project-name",
					},
				},
			},
		},
		{
			name:    "prepareUserGrantsQuery sql err",
			prepare: prepareUserGrantsQuery,
			want: want{
				sqlExpectations: mockQueryErr(
					userGrantsStmt,
					sql.ErrConnDone,
				),
				err: func(err error) (error, bool) {
					if !errors.Is(err, sql.ErrConnDone) {
						return fmt.Errorf("err should be sql.ErrConnDone got: %w", err), false
					}
					return nil, true
				},
			},
			object: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPrepare(t, tt.prepare, tt.object, tt.want.sqlExpectations, tt.want.err)
		})
	}
}
