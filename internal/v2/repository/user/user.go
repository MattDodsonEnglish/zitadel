package user

import (
	"context"
	"encoding/json"
	"github.com/caos/zitadel/internal/errors"
	"github.com/caos/zitadel/internal/eventstore/v2"
	"github.com/caos/zitadel/internal/eventstore/v2/repository"
	"time"
)

const (
	UniqueUsername            = "usernames"
	userEventTypePrefix       = eventstore.EventType("user.")
	UserLockedType            = userEventTypePrefix + "locked"
	UserUnlockedType          = userEventTypePrefix + "unlocked"
	UserDeactivatedType       = userEventTypePrefix + "deactivated"
	UserReactivatedType       = userEventTypePrefix + "reactivated"
	UserRemovedType           = userEventTypePrefix + "removed"
	UserTokenAddedType        = userEventTypePrefix + "token.added"
	UserDomainClaimedType     = userEventTypePrefix + "domain.claimed"
	UserDomainClaimedSentType = userEventTypePrefix + "domain.claimed.sent"
	UserUserNameChangedType   = userEventTypePrefix + "username.changed"
)

func NewAddUsernameUniqueConstraint(userName, resourceOwner string, userLoginMustBeDomain bool) *eventstore.EventUniqueConstraint {
	uniqueUserName := userName
	if userLoginMustBeDomain {
		uniqueUserName = userName + resourceOwner
	}
	return eventstore.NewAddEventUniqueConstraint(
		UniqueUsername,
		uniqueUserName,
		"Errors.User.AlreadyExists")
}

func NewRemoveUsernameUniqueConstraint(userName, resourceOwner string, userLoginMustBeDomain bool) *eventstore.EventUniqueConstraint {
	uniqueUserName := userName
	if userLoginMustBeDomain {
		uniqueUserName = userName + resourceOwner
	}
	return eventstore.NewRemoveEventUniqueConstraint(
		UniqueUsername,
		uniqueUserName)
}

type UserLockedEvent struct {
	eventstore.BaseEvent `json:"-"`
}

func (e *UserLockedEvent) Data() interface{} {
	return nil
}

func (e *UserLockedEvent) UniqueConstraints() []*eventstore.EventUniqueConstraint {
	return nil
}

func NewUserLockedEvent(ctx context.Context) *UserLockedEvent {
	return &UserLockedEvent{
		BaseEvent: *eventstore.NewBaseEventForPush(
			ctx,
			UserLockedType,
		),
	}
}

func UserLockedEventMapper(event *repository.Event) (eventstore.EventReader, error) {
	return &UserLockedEvent{
		BaseEvent: *eventstore.BaseEventFromRepo(event),
	}, nil
}

type UserUnlockedEvent struct {
	eventstore.BaseEvent `json:"-"`
}

func (e *UserUnlockedEvent) Data() interface{} {
	return nil
}

func (e *UserUnlockedEvent) UniqueConstraints() []*eventstore.EventUniqueConstraint {
	return nil
}

func NewUserUnlockedEvent(ctx context.Context) *UserUnlockedEvent {
	return &UserUnlockedEvent{
		BaseEvent: *eventstore.NewBaseEventForPush(
			ctx,
			UserUnlockedType,
		),
	}
}

func UserUnlockedEventMapper(event *repository.Event) (eventstore.EventReader, error) {
	return &UserUnlockedEvent{
		BaseEvent: *eventstore.BaseEventFromRepo(event),
	}, nil
}

type UserDeactivatedEvent struct {
	eventstore.BaseEvent `json:"-"`
}

func (e *UserDeactivatedEvent) Data() interface{} {
	return nil
}

func (e *UserDeactivatedEvent) UniqueConstraints() []*eventstore.EventUniqueConstraint {
	return nil
}

func NewUserDeactivatedEvent(ctx context.Context) *UserDeactivatedEvent {
	return &UserDeactivatedEvent{
		BaseEvent: *eventstore.NewBaseEventForPush(
			ctx,
			UserDeactivatedType,
		),
	}
}

func UserDeactivatedEventMapper(event *repository.Event) (eventstore.EventReader, error) {
	return &UserDeactivatedEvent{
		BaseEvent: *eventstore.BaseEventFromRepo(event),
	}, nil
}

type UserReactivatedEvent struct {
	eventstore.BaseEvent `json:"-"`
}

func (e *UserReactivatedEvent) Data() interface{} {
	return nil
}

func (e *UserReactivatedEvent) UniqueConstraints() []*eventstore.EventUniqueConstraint {
	return nil
}

func NewUserReactivatedEvent(ctx context.Context) *UserReactivatedEvent {
	return &UserReactivatedEvent{
		BaseEvent: *eventstore.NewBaseEventForPush(
			ctx,
			UserReactivatedType,
		),
	}
}

func UserReactivatedEventMapper(event *repository.Event) (eventstore.EventReader, error) {
	return &UserReactivatedEvent{
		BaseEvent: *eventstore.BaseEventFromRepo(event),
	}, nil
}

type UserRemovedEvent struct {
	eventstore.BaseEvent `json:"-"`

	UserName              string
	UserLoginMustBeDomain bool
}

func (e *UserRemovedEvent) Data() interface{} {
	return nil
}

func (e *UserRemovedEvent) UniqueConstraints() []*eventstore.EventUniqueConstraint {
	return []*eventstore.EventUniqueConstraint{NewRemoveUsernameUniqueConstraint(e.UserName, e.ResourceOwner(), e.UserLoginMustBeDomain)}
}

func NewUserRemovedEvent(ctx context.Context, resourceOwner, userName string, userLoginMustBeDomain bool) *UserRemovedEvent {
	return &UserRemovedEvent{
		BaseEvent: *eventstore.NewBaseEventForPushWithResourceOwner(
			ctx,
			UserRemovedType,
			resourceOwner,
		),
		UserName:              userName,
		UserLoginMustBeDomain: userLoginMustBeDomain,
	}
}

func UserRemovedEventMapper(event *repository.Event) (eventstore.EventReader, error) {
	return &UserRemovedEvent{
		BaseEvent: *eventstore.BaseEventFromRepo(event),
	}, nil
}

type UserTokenAddedEvent struct {
	eventstore.BaseEvent `json:"-"`

	TokenID           string    `json:"tokenId"`
	ApplicationID     string    `json:"applicationId"`
	UserAgentID       string    `json:"userAgentId"`
	Audience          []string  `json:"audience"`
	Scopes            []string  `json:"scopes""`
	Expiration        time.Time `json:"expiration"`
	PreferredLanguage string    `json:"preferredLanguage"`
}

func (e *UserTokenAddedEvent) Data() interface{} {
	return e
}

func (e *UserTokenAddedEvent) UniqueConstraints() []*eventstore.EventUniqueConstraint {
	return nil
}

func NewUserTokenAddedEvent(
	ctx context.Context,
	tokenID,
	applicationID,
	userAgentID,
	preferredLanguage string,
	audience,
	scopes []string,
	expiration time.Time,
) *UserTokenAddedEvent {
	return &UserTokenAddedEvent{
		BaseEvent: *eventstore.NewBaseEventForPush(
			ctx,
			UserTokenAddedType,
		),
		TokenID:           tokenID,
		ApplicationID:     applicationID,
		UserAgentID:       userAgentID,
		Audience:          audience,
		Scopes:            scopes,
		Expiration:        expiration,
		PreferredLanguage: preferredLanguage,
	}
}

func UserTokenAddedEventMapper(event *repository.Event) (eventstore.EventReader, error) {
	tokenAdded := &UserTokenAddedEvent{
		BaseEvent: *eventstore.BaseEventFromRepo(event),
	}
	err := json.Unmarshal(event.Data, tokenAdded)
	if err != nil {
		return nil, errors.ThrowInternal(err, "USER-7M9sd", "unable to unmarshal token added")
	}

	return tokenAdded, nil
}

type DomainClaimedEvent struct {
	eventstore.BaseEvent `json:"-"`

	UserName              string `json:"userName"`
	oldUserName           string `json:"-"`
	userLoginMustBeDomain bool   `json:"-"`
}

func (e *DomainClaimedEvent) Data() interface{} {
	return e
}

func (e *DomainClaimedEvent) UniqueConstraints() []*eventstore.EventUniqueConstraint {
	return []*eventstore.EventUniqueConstraint{
		NewRemoveUsernameUniqueConstraint(e.oldUserName, e.ResourceOwner(), e.userLoginMustBeDomain),
		NewAddUsernameUniqueConstraint(e.UserName, e.ResourceOwner(), e.userLoginMustBeDomain),
	}
}

func NewDomainClaimedEvent(
	ctx context.Context,
	userName,
	oldUserName string,
	userLoginMustBeDomain bool,
) *DomainClaimedEvent {
	return &DomainClaimedEvent{
		BaseEvent: *eventstore.NewBaseEventForPush(
			ctx,
			UserDomainClaimedType,
		),
		UserName:              userName,
		oldUserName:           oldUserName,
		userLoginMustBeDomain: userLoginMustBeDomain,
	}
}

func DomainClaimedEventMapper(event *repository.Event) (eventstore.EventReader, error) {
	domainClaimed := &DomainClaimedEvent{
		BaseEvent: *eventstore.BaseEventFromRepo(event),
	}
	err := json.Unmarshal(event.Data, domainClaimed)
	if err != nil {
		return nil, errors.ThrowInternal(err, "USER-aR8jc", "unable to unmarshal domain claimed")
	}

	return domainClaimed, nil
}

type DomainClaimedSentEvent struct {
	eventstore.BaseEvent `json:"-"`
}

func (e *DomainClaimedSentEvent) Data() interface{} {
	return nil
}

func (e *DomainClaimedSentEvent) UniqueConstraints() []*eventstore.EventUniqueConstraint {
	return nil
}

func NewDomainClaimedSentEvent(
	ctx context.Context,
) *DomainClaimedSentEvent {
	return &DomainClaimedSentEvent{
		BaseEvent: *eventstore.NewBaseEventForPush(
			ctx,
			UserDomainClaimedSentType,
		),
	}
}

func DomainClaimedSentEventMapper(event *repository.Event) (eventstore.EventReader, error) {
	return &DomainClaimedSentEvent{
		BaseEvent: *eventstore.BaseEventFromRepo(event),
	}, nil
}

type UsernameChangedEvent struct {
	eventstore.BaseEvent `json:"-"`

	UserName              string `json:"userName"`
	oldUserName           string `json:"-"`
	userLoginMustBeDomain bool   `json:"-"`
}

func (e *UsernameChangedEvent) Data() interface{} {
	return e
}

func (e *UsernameChangedEvent) UniqueConstraints() []*eventstore.EventUniqueConstraint {
	return []*eventstore.EventUniqueConstraint{
		NewRemoveUsernameUniqueConstraint(e.oldUserName, e.ResourceOwner(), e.userLoginMustBeDomain),
		NewAddUsernameUniqueConstraint(e.UserName, e.ResourceOwner(), e.userLoginMustBeDomain),
	}
}

func NewUsernameChangedEvent(
	ctx context.Context,
	oldUserName,
	newUserName string,
	userLoginMustBeDomain bool,
) *UsernameChangedEvent {
	return &UsernameChangedEvent{
		BaseEvent: *eventstore.NewBaseEventForPush(
			ctx,
			UserUserNameChangedType,
		),
		UserName:              newUserName,
		oldUserName:           oldUserName,
		userLoginMustBeDomain: userLoginMustBeDomain,
	}
}

func UsernameChangedEventMapper(event *repository.Event) (eventstore.EventReader, error) {
	domainClaimed := &UsernameChangedEvent{
		BaseEvent: *eventstore.BaseEventFromRepo(event),
	}
	err := json.Unmarshal(event.Data, domainClaimed)
	if err != nil {
		return nil, errors.ThrowInternal(err, "USER-4Bm9s", "unable to unmarshal username changed")
	}

	return domainClaimed, nil
}