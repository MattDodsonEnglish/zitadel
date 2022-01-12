package query

import (
	"context"
	"database/sql"
	errs "errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"

	"github.com/caos/zitadel/internal/domain"
	"github.com/caos/zitadel/internal/errors"
	"github.com/caos/zitadel/internal/query/projection"
)

type UserGrant struct {
	ID           string
	CreationDate time.Time
	ChangeDate   time.Time
	Sequence     uint64
	Roles        []string
	GrantID      string
	State        domain.UserGrantState

	UserID            string
	Username          string
	UserType          domain.UserType
	UserResourceOwner string
	FirstName         string
	LastName          string
	Email             string
	DisplayName       string
	AvatarURL         string

	ResourceOwner    string
	OrgName          string
	OrgPrimaryDomain string

	ProjectID   string
	ProjectName string
}

type UserGrants struct {
	SearchResponse
	UserGrants []*UserGrant
}

type UserGrantsQueries struct {
	SearchRequest
	Queries []SearchQuery
}

func (q *UserGrantsQueries) toQuery(query sq.SelectBuilder) sq.SelectBuilder {
	query = q.SearchRequest.toQuery(query)
	for _, q := range q.Queries {
		query = q.toQuery(query)
	}
	return query
}

func NewUserGrantUserIDSearchQuery(id string) (SearchQuery, error) {
	return NewTextQuery(UserGrantUserID, id, TextEquals)
}

func NewUserGrantProjectIDSearchQuery(id string) (SearchQuery, error) {
	return NewTextQuery(UserGrantProjectID, id, TextEquals)
}

func NewUserGrantProjectNameSearchQuery(id string, method TextComparison) (SearchQuery, error) {
	return NewTextQuery(UserGrantProjectID, id, method)
}

func NewUserGrantProjectOwnerSearchQuery(id string) (SearchQuery, error) {
	return NewTextQuery(ProjectColumnResourceOwner, id, TextEquals)
}

func NewUserGrantResourceOwnerSearchQuery(id string) (SearchQuery, error) {
	return NewTextQuery(UserGrantResourceOwner, id, TextEquals)
}

func NewUserGrantGrantIDSearchQuery(id string) (SearchQuery, error) {
	return NewTextQuery(UserGrantGrantID, id, TextEquals)
}

func NewUserGrantUserTypeQuery(typ domain.UserType) (SearchQuery, error) {
	return NewNumberQuery(UserTypeCol, typ, NumberEquals)
}

func NewUserGrantDisplayNameQuery(displayName string, method TextComparison) (SearchQuery, error) {
	return NewTextQuery(HumanDisplayNameCol, displayName, method)
}

func NewUserGrantEmailQuery(email string, method TextComparison) (SearchQuery, error) {
	return NewTextQuery(HumanEmailCol, email, method)
}

func NewUserGrantFirstNameQuery(value string, method TextComparison) (SearchQuery, error) {
	return NewTextQuery(HumanFirstNameCol, value, method)
}

func NewUserGrantLastNameQuery(value string, method TextComparison) (SearchQuery, error) {
	return NewTextQuery(HumanLastNameCol, value, method)
}

func NewUserGrantUsernameQuery(value string, method TextComparison) (SearchQuery, error) {
	return NewTextQuery(UserUsernameCol, value, method)
}

func NewUserGrantDomainQuery(value string, method TextComparison) (SearchQuery, error) {
	return NewTextQuery(OrgColumnDomain, value, method)
}

func NewUserGrantOrgNameQuery(value string, method TextComparison) (SearchQuery, error) {
	return NewTextQuery(OrgColumnName, value, method)
}

func NewUserGrantProjectNameQuery(value string, method TextComparison) (SearchQuery, error) {
	return NewTextQuery(ProjectColumnName, value, method)
}

func NewUserGrantRoleQuery(value string) (SearchQuery, error) {
	return NewTextQuery(UserGrantRoles, value, TextListContains)
}

func NewUserGrantWithGrantedQuery(owner string) (SearchQuery, error) {
	orgQuery, err := NewUserGrantResourceOwnerSearchQuery(owner)
	if err != nil {
		return nil, err
	}
	projectQuery, err := NewUserGrantProjectOwnerSearchQuery(owner)
	if err != nil {
		return nil, err
	}
	return newOrQuery(orgQuery, projectQuery)
}

func NewUserGrantContainsRolesSearchQuery(roles ...string) (SearchQuery, error) {
	r := make([]interface{}, len(roles))
	for i, role := range roles {
		r[i] = role
	}
	return NewListQuery(UserGrantRoles, r, ListIn)
}

var (
	userGrantTable = table{
		name: projection.UserGrantProjectionTable,
	}
	UserGrantID = Column{
		name:  projection.UserGrantID,
		table: userGrantTable,
	}
	UserGrantResourceOwner = Column{
		name:  projection.UserGrantResourceOwner,
		table: userGrantTable,
	}
	UserGrantCreationDate = Column{
		name:  projection.UserGrantCreationDate,
		table: userGrantTable,
	}
	UserGrantChangeDate = Column{
		name:  projection.UserGrantChangeDate,
		table: userGrantTable,
	}
	UserGrantSequence = Column{
		name:  projection.UserGrantSequence,
		table: userGrantTable,
	}
	UserGrantUserID = Column{
		name:  projection.UserGrantUserID,
		table: userGrantTable,
	}
	UserGrantProjectID = Column{
		name:  projection.UserGrantProjectID,
		table: userGrantTable,
	}
	UserGrantGrantID = Column{
		name:  projection.UserGrantGrantID,
		table: userGrantTable,
	}
	UserGrantRoles = Column{
		name:  projection.UserGrantRoles,
		table: userGrantTable,
	}
	UserGrantState = Column{
		name:  projection.UserGrantState,
		table: userGrantTable,
	}
)

func (q *Queries) UserGrantByID(ctx context.Context, id string) (*UserGrant, error) {
	stmt, scan := prepareUserGrantQuery()
	query, args, err := stmt.Where(sq.Eq{
		UserGrantID.identifier(): id,
	}).ToSql()
	if err != nil {
		return nil, errors.ThrowInternal(err, "QUERY-Fa1KW", "Errors.Query.SQLStatement")
	}

	row := q.client.QueryRowContext(ctx, query, args...)
	return scan(row)
}

func (q *Queries) UserGrants(ctx context.Context, queries *UserGrantsQueries) (*UserGrants, error) {
	query, scan := prepareUserGrantsQuery()
	stmt, args, err := queries.toQuery(query).ToSql()
	if err != nil {
		return nil, errors.ThrowInternal(err, "QUERY-wXnQR", "Errors.Query.SQLStatement")
	}

	latestSequence, err := q.latestSequence(ctx, userGrantTable)
	if err != nil {
		return nil, err
	}

	rows, err := q.client.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	grants, err := scan(rows)
	if err != nil {
		return nil, err
	}

	grants.LatestSequence = latestSequence
	return grants, nil
}

func prepareUserGrantQuery() (sq.SelectBuilder, func(*sql.Row) (*UserGrant, error)) {
	return sq.Select(
			UserGrantID.identifier(),
			UserGrantCreationDate.identifier(),
			UserGrantChangeDate.identifier(),
			UserGrantSequence.identifier(),
			UserGrantGrantID.identifier(),
			UserGrantRoles.identifier(),
			UserGrantState.identifier(),

			UserGrantUserID.identifier(),
			UserUsernameCol.identifier(),
			UserTypeCol.identifier(),
			UserResourceOwnerCol.identifier(),
			HumanFirstNameCol.identifier(),
			HumanLastNameCol.identifier(),
			HumanEmailCol.identifier(),
			HumanDisplayNameCol.identifier(),
			HumanAvaterURLCol.identifier(),

			UserGrantResourceOwner.identifier(),
			OrgColumnName.identifier(),
			OrgColumnDomain.identifier(),

			UserGrantProjectID.identifier(),
			ProjectColumnName.identifier(),
		).
			From(userGrantTable.identifier()).
			LeftJoin(join(UserIDCol, UserGrantUserID)).
			LeftJoin(join(HumanUserIDCol, UserGrantUserID)).
			LeftJoin(join(OrgColumnID, UserGrantResourceOwner)).
			LeftJoin(join(ProjectColumnID, UserGrantProjectID)).
			PlaceholderFormat(sq.Dollar),
		func(row *sql.Row) (*UserGrant, error) {
			g := new(UserGrant)

			var (
				roles       = pq.StringArray{}
				username    sql.NullString
				firstName   sql.NullString
				userType    sql.NullInt32
				userOwner   sql.NullString
				lastName    sql.NullString
				email       sql.NullString
				displayName sql.NullString
				avatarURL   sql.NullString

				orgName   sql.NullString
				orgDomain sql.NullString

				projectName sql.NullString
			)

			err := row.Scan(
				&g.ID,
				&g.CreationDate,
				&g.ChangeDate,
				&g.Sequence,
				&g.GrantID,
				&roles,
				&g.State,

				&g.UserID,
				&username,
				&userType,
				&userOwner,
				&firstName,
				&lastName,
				&email,
				&displayName,
				&avatarURL,

				&g.ResourceOwner,
				&orgName,
				&orgDomain,

				&g.ProjectID,
				&projectName,
			)
			if err != nil {
				if errs.Is(err, sql.ErrNoRows) {
					return nil, errors.ThrowNotFound(err, "QUERY-wIPkA", "Errors.UserGrant.NotFound")
				}
				return nil, errors.ThrowInternal(err, "QUERY-oQPcP", "Errors.Internal")
			}

			g.Roles = roles
			g.Username = username.String
			g.UserType = domain.UserType(userType.Int32)
			g.UserResourceOwner = userOwner.String
			g.FirstName = firstName.String
			g.LastName = lastName.String
			g.Email = email.String
			g.DisplayName = displayName.String
			g.AvatarURL = avatarURL.String
			g.OrgName = orgName.String
			g.OrgPrimaryDomain = orgDomain.String
			g.ProjectName = projectName.String

			return g, nil
		}
}

func prepareUserGrantsQuery() (sq.SelectBuilder, func(*sql.Rows) (*UserGrants, error)) {
	return sq.Select(
			UserGrantID.identifier(),
			UserGrantCreationDate.identifier(),
			UserGrantChangeDate.identifier(),
			UserGrantSequence.identifier(),
			UserGrantGrantID.identifier(),
			UserGrantRoles.identifier(),
			UserGrantState.identifier(),

			UserGrantUserID.identifier(),
			UserUsernameCol.identifier(),
			UserTypeCol.identifier(),
			UserResourceOwnerCol.identifier(),
			HumanFirstNameCol.identifier(),
			HumanLastNameCol.identifier(),
			HumanEmailCol.identifier(),
			HumanDisplayNameCol.identifier(),
			HumanAvaterURLCol.identifier(),

			UserGrantResourceOwner.identifier(),
			OrgColumnName.identifier(),
			OrgColumnDomain.identifier(),

			UserGrantProjectID.identifier(),
			ProjectColumnName.identifier(),

			countColumn.identifier(),
		).
			From(userGrantTable.identifier()).
			LeftJoin(join(UserIDCol, UserGrantUserID)).
			LeftJoin(join(HumanUserIDCol, UserGrantUserID)).
			LeftJoin(join(OrgColumnID, UserGrantResourceOwner)).
			LeftJoin(join(ProjectColumnID, UserGrantProjectID)).
			PlaceholderFormat(sq.Dollar),
		func(rows *sql.Rows) (*UserGrants, error) {
			userGrants := make([]*UserGrant, 0)
			var count uint64
			for rows.Next() {
				g := new(UserGrant)

				var (
					roles       = pq.StringArray{}
					username    sql.NullString
					userType    sql.NullInt32
					userOwner   sql.NullString
					firstName   sql.NullString
					lastName    sql.NullString
					email       sql.NullString
					displayName sql.NullString
					avatarURL   sql.NullString

					orgName   sql.NullString
					orgDomain sql.NullString

					projectName sql.NullString
				)

				err := rows.Scan(
					&g.ID,
					&g.CreationDate,
					&g.ChangeDate,
					&g.Sequence,
					&g.GrantID,
					&roles,
					&g.State,

					&g.UserID,
					&username,
					&userType,
					&userOwner,
					&firstName,
					&lastName,
					&email,
					&displayName,
					&avatarURL,

					&g.ResourceOwner,
					&orgName,
					&orgDomain,

					&g.ProjectID,
					&projectName,

					&count,
				)
				if err != nil {
					return nil, err
				}

				g.Roles = roles
				g.Username = username.String
				g.UserType = domain.UserType(userType.Int32)
				g.UserResourceOwner = userOwner.String
				g.FirstName = firstName.String
				g.LastName = lastName.String
				g.Email = email.String
				g.DisplayName = displayName.String
				g.AvatarURL = avatarURL.String
				g.OrgName = orgName.String
				g.OrgPrimaryDomain = orgDomain.String
				g.ProjectName = projectName.String

				userGrants = append(userGrants, g)
			}

			if err := rows.Close(); err != nil {
				return nil, errors.ThrowInternal(err, "QUERY-iGvmP", "Errors.Query.CloseRows")
			}

			return &UserGrants{
				UserGrants: userGrants,
				SearchResponse: SearchResponse{
					Count: count,
				},
			}, nil
		}
}
