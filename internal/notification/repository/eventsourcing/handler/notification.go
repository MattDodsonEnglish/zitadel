package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/caos/logging"
	"github.com/caos/zitadel/internal/api/authz"
	"github.com/caos/zitadel/internal/command"
	sd "github.com/caos/zitadel/internal/config/systemdefaults"
	"github.com/caos/zitadel/internal/crypto"
	"github.com/caos/zitadel/internal/domain"
	"github.com/caos/zitadel/internal/errors"
	v1 "github.com/caos/zitadel/internal/eventstore/v1"
	"github.com/caos/zitadel/internal/eventstore/v1/models"
	queryv1 "github.com/caos/zitadel/internal/eventstore/v1/query"
	"github.com/caos/zitadel/internal/eventstore/v1/spooler"
	"github.com/caos/zitadel/internal/i18n"
	"github.com/caos/zitadel/internal/notification/types"
	"github.com/caos/zitadel/internal/query"
	user_repo "github.com/caos/zitadel/internal/repository/user"
	es_model "github.com/caos/zitadel/internal/user/repository/eventsourcing/model"
	"github.com/caos/zitadel/internal/user/repository/view"
	"github.com/caos/zitadel/internal/user/repository/view/model"
)

const (
	notificationTable = "notification.notifications"
	NotifyUserID      = "NOTIFICATION"
)

type Notification struct {
	handler
	command        *command.Commands
	systemDefaults sd.SystemDefaults
	AesCrypto      crypto.EncryptionAlgorithm
	statikDir      http.FileSystem
	subscription   *v1.Subscription
	apiDomain      string
	queries        *query.Queries
}

func newNotification(
	handler handler,
	command *command.Commands,
	query *query.Queries,
	defaults sd.SystemDefaults,
	aesCrypto crypto.EncryptionAlgorithm,
	statikDir http.FileSystem,
	apiDomain string,
) *Notification {
	h := &Notification{
		handler:        handler,
		command:        command,
		systemDefaults: defaults,
		statikDir:      statikDir,
		AesCrypto:      aesCrypto,
		apiDomain:      apiDomain,
		queries:        query,
	}

	h.subscribe()

	return h
}

func (k *Notification) subscribe() {
	k.subscription = k.es.Subscribe(k.AggregateTypes()...)
	go func() {
		for event := range k.subscription.Events {
			queryv1.ReduceEvent(k, event)
		}
	}()
}

func (n *Notification) ViewModel() string {
	return notificationTable
}

func (n *Notification) Subscription() *v1.Subscription {
	return n.subscription
}

func (_ *Notification) AggregateTypes() []models.AggregateType {
	return []models.AggregateType{es_model.UserAggregate}
}

func (n *Notification) CurrentSequence() (uint64, error) {
	sequence, err := n.view.GetLatestNotificationSequence()
	if err != nil {
		return 0, err
	}
	return sequence.CurrentSequence, nil
}

func (n *Notification) EventQuery() (*models.SearchQuery, error) {
	sequence, err := n.view.GetLatestNotificationSequence()
	if err != nil {
		return nil, err
	}
	return view.UserQuery(sequence.CurrentSequence), nil
}

func (n *Notification) Reduce(event *models.Event) (err error) {
	switch event.Type {
	case es_model.InitializedUserCodeAdded,
		es_model.InitializedHumanCodeAdded:
		err = n.handleInitUserCode(event)
	case es_model.UserEmailCodeAdded,
		es_model.HumanEmailCodeAdded:
		err = n.handleEmailVerificationCode(event)
	case es_model.UserPhoneCodeAdded,
		es_model.HumanPhoneCodeAdded:
		err = n.handlePhoneVerificationCode(event)
	case es_model.UserPasswordCodeAdded,
		es_model.HumanPasswordCodeAdded:
		err = n.handlePasswordCode(event)
	case es_model.DomainClaimed:
		err = n.handleDomainClaimed(event)
	case models.EventType(user_repo.HumanPasswordlessInitCodeRequestedType):
		err = n.handlePasswordlessRegistrationLink(event)
	}
	if err != nil {
		return err
	}
	return n.view.ProcessedNotificationSequence(event)
}

func (n *Notification) handleInitUserCode(event *models.Event) (err error) {
	initCode := new(es_model.InitUserCode)
	if err := initCode.SetData(event); err != nil {
		return err
	}
	alreadyHandled, err := n.checkIfCodeAlreadyHandledOrExpired(event, initCode.Expiry,
		es_model.InitializedUserCodeAdded, es_model.InitializedUserCodeSent,
		es_model.InitializedHumanCodeAdded, es_model.InitializedHumanCodeSent)
	if err != nil || alreadyHandled {
		return err
	}
	ctx := getSetNotifyContextData(event.ResourceOwner)
	colors, err := n.getLabelPolicy(ctx)
	if err != nil {
		return err
	}

	template, err := n.getMailTemplate(ctx)
	if err != nil {
		return err
	}

	user, err := n.getUserByID(event.AggregateID)
	if err != nil {
		return err
	}

	translator, err := n.getTranslatorWithOrgTexts(user.ResourceOwner, domain.InitCodeMessageType)
	if err != nil {
		return err
	}

	err = types.SendUserInitCode(string(template.Template), translator, user, initCode, n.systemDefaults, n.AesCrypto, colors, n.apiDomain)
	if err != nil {
		return err
	}
	return n.command.HumanInitCodeSent(ctx, event.ResourceOwner, event.AggregateID)
}

func (n *Notification) handlePasswordCode(event *models.Event) (err error) {
	pwCode := new(es_model.PasswordCode)
	if err := pwCode.SetData(event); err != nil {
		return err
	}
	alreadyHandled, err := n.checkIfCodeAlreadyHandledOrExpired(event, pwCode.Expiry,
		es_model.UserPasswordCodeAdded, es_model.UserPasswordCodeSent,
		es_model.HumanPasswordCodeAdded, es_model.HumanPasswordCodeSent)
	if err != nil || alreadyHandled {
		return err
	}
	ctx := getSetNotifyContextData(event.ResourceOwner)
	colors, err := n.getLabelPolicy(ctx)
	if err != nil {
		return err
	}

	template, err := n.getMailTemplate(ctx)
	if err != nil {
		return err
	}

	user, err := n.getUserByID(event.AggregateID)
	if err != nil {
		return err
	}

	translator, err := n.getTranslatorWithOrgTexts(user.ResourceOwner, domain.PasswordResetMessageType)
	if err != nil {
		return err
	}
	err = types.SendPasswordCode(string(template.Template), translator, user, pwCode, n.systemDefaults, n.AesCrypto, colors, n.apiDomain)
	if err != nil {
		return err
	}
	return n.command.PasswordCodeSent(ctx, event.ResourceOwner, event.AggregateID)
}

func (n *Notification) handleEmailVerificationCode(event *models.Event) (err error) {
	emailCode := new(es_model.EmailCode)
	if err := emailCode.SetData(event); err != nil {
		return err
	}
	alreadyHandled, err := n.checkIfCodeAlreadyHandledOrExpired(event, emailCode.Expiry,
		es_model.UserEmailCodeAdded, es_model.UserEmailCodeSent,
		es_model.HumanEmailCodeAdded, es_model.HumanEmailCodeSent)
	if err != nil || alreadyHandled {
		return nil
	}
	ctx := getSetNotifyContextData(event.ResourceOwner)
	colors, err := n.getLabelPolicy(ctx)
	if err != nil {
		return err
	}

	template, err := n.getMailTemplate(ctx)
	if err != nil {
		return err
	}

	user, err := n.getUserByID(event.AggregateID)
	if err != nil {
		return err
	}

	translator, err := n.getTranslatorWithOrgTexts(user.ResourceOwner, domain.VerifyEmailMessageType)
	if err != nil {
		return err
	}

	err = types.SendEmailVerificationCode(string(template.Template), translator, user, emailCode, n.systemDefaults, n.AesCrypto, colors, n.apiDomain)
	if err != nil {
		return err
	}
	return n.command.HumanEmailVerificationCodeSent(ctx, event.ResourceOwner, event.AggregateID)
}

func (n *Notification) handlePhoneVerificationCode(event *models.Event) (err error) {
	phoneCode := new(es_model.PhoneCode)
	if err := phoneCode.SetData(event); err != nil {
		return err
	}
	alreadyHandled, err := n.checkIfCodeAlreadyHandledOrExpired(event, phoneCode.Expiry,
		es_model.UserPhoneCodeAdded, es_model.UserPhoneCodeSent,
		es_model.HumanPhoneCodeAdded, es_model.HumanPhoneCodeSent)
	if err != nil || alreadyHandled {
		return nil
	}
	user, err := n.getUserByID(event.AggregateID)
	if err != nil {
		return err
	}
	translator, err := n.getTranslatorWithOrgTexts(user.ResourceOwner, domain.VerifyPhoneMessageType)
	if err != nil {
		return err
	}
	err = types.SendPhoneVerificationCode(translator, user, phoneCode, n.systemDefaults, n.AesCrypto)
	if err != nil {
		return err
	}
	return n.command.HumanPhoneVerificationCodeSent(getSetNotifyContextData(event.ResourceOwner), event.ResourceOwner, event.AggregateID)
}

func (n *Notification) handleDomainClaimed(event *models.Event) (err error) {
	alreadyHandled, err := n.checkIfAlreadyHandled(event.AggregateID, event.Sequence, es_model.DomainClaimed, es_model.DomainClaimedSent)
	if err != nil || alreadyHandled {
		return nil
	}
	data := make(map[string]string)
	if err := json.Unmarshal(event.Data, &data); err != nil {
		logging.Log("HANDLE-Gghq2").WithError(err).Error("could not unmarshal event data")
		return errors.ThrowInternal(err, "HANDLE-7hgj3", "could not unmarshal event")
	}
	user, err := n.getUserByID(event.AggregateID)
	if err != nil {
		return err
	}
	if user.LastEmail == "" {
		return nil
	}
	ctx := getSetNotifyContextData(event.ResourceOwner)
	colors, err := n.getLabelPolicy(ctx)
	if err != nil {
		return err
	}

	template, err := n.getMailTemplate(ctx)
	if err != nil {
		return err
	}

	translator, err := n.getTranslatorWithOrgTexts(user.ResourceOwner, domain.DomainClaimedMessageType)
	if err != nil {
		return err
	}
	err = types.SendDomainClaimed(string(template.Template), translator, user, data["userName"], n.systemDefaults, colors, n.apiDomain)
	if err != nil {
		return err
	}
	return n.command.UserDomainClaimedSent(ctx, event.ResourceOwner, event.AggregateID)
}

func (n *Notification) handlePasswordlessRegistrationLink(event *models.Event) (err error) {
	addedEvent := new(user_repo.HumanPasswordlessInitCodeRequestedEvent)
	if err := json.Unmarshal(event.Data, addedEvent); err != nil {
		return err
	}
	events, err := n.getUserEvents(event.AggregateID, event.Sequence)
	if err != nil {
		return err
	}
	for _, e := range events {
		if e.Type == models.EventType(user_repo.HumanPasswordlessInitCodeSentType) {
			sentEvent := new(user_repo.HumanPasswordlessInitCodeSentEvent)
			if err := json.Unmarshal(e.Data, sentEvent); err != nil {
				return err
			}
			if sentEvent.ID == addedEvent.ID {
				return nil
			}
		}
	}
	user, err := n.getUserByID(event.AggregateID)
	if err != nil {
		return err
	}
	ctx := getSetNotifyContextData(event.ResourceOwner)
	colors, err := n.getLabelPolicy(ctx)
	if err != nil {
		return err
	}

	template, err := n.getMailTemplate(ctx)
	if err != nil {
		return err
	}

	translator, err := n.getTranslatorWithOrgTexts(user.ResourceOwner, domain.PasswordlessRegistrationMessageType)
	if err != nil {
		return err
	}
	err = types.SendPasswordlessRegistrationLink(string(template.Template), translator, user, addedEvent, n.systemDefaults, n.AesCrypto, colors, n.apiDomain)
	if err != nil {
		return err
	}
	return n.command.HumanPasswordlessInitCodeSent(ctx, event.AggregateID, event.ResourceOwner, addedEvent.ID)
}

func (n *Notification) checkIfCodeAlreadyHandledOrExpired(event *models.Event, expiry time.Duration, eventTypes ...models.EventType) (bool, error) {
	if event.CreationDate.Add(expiry).Before(time.Now().UTC()) {
		return true, nil
	}
	return n.checkIfAlreadyHandled(event.AggregateID, event.Sequence, eventTypes...)
}

func (n *Notification) checkIfAlreadyHandled(userID string, sequence uint64, eventTypes ...models.EventType) (bool, error) {
	events, err := n.getUserEvents(userID, sequence)
	if err != nil {
		return false, err
	}
	for _, event := range events {
		for _, eventType := range eventTypes {
			if event.Type == eventType {
				return true, nil
			}
		}
	}
	return false, nil
}

func (n *Notification) getUserEvents(userID string, sequence uint64) ([]*models.Event, error) {
	query, err := view.UserByIDQuery(userID, sequence)
	if err != nil {
		return nil, err
	}

	return n.es.FilterEvents(context.Background(), query)
}

func (n *Notification) OnError(event *models.Event, err error) error {
	logging.LogWithFields("SPOOL-s9opc", "id", event.AggregateID, "sequence", event.Sequence).WithError(err).Warn("something went wrong in notification handler")
	return spooler.HandleError(event, err, n.view.GetLatestNotificationFailedEvent, n.view.ProcessedNotificationFailedEvent, n.view.ProcessedNotificationSequence, n.errorCountUntilSkip)
}

func (n *Notification) OnSuccess() error {
	return spooler.HandleSuccess(n.view.UpdateNotificationSpoolerRunTimestamp)
}

func getSetNotifyContextData(orgID string) context.Context {
	return authz.SetCtxData(context.Background(), authz.CtxData{UserID: NotifyUserID, OrgID: orgID})
}

// Read organization specific colors
func (n *Notification) getLabelPolicy(ctx context.Context) (*query.LabelPolicy, error) {
	return n.queries.ActiveLabelPolicyByOrg(ctx, authz.GetCtxData(ctx).OrgID)
}

// Read organization specific template
func (n *Notification) getMailTemplate(ctx context.Context) (*query.MailTemplate, error) {
	return n.queries.MailTemplateByOrg(ctx, authz.GetCtxData(ctx).OrgID)
}

func (n *Notification) getTranslatorWithOrgTexts(orgID, textType string) (*i18n.Translator, error) {
	translator, err := i18n.NewTranslator(n.statikDir, i18n.TranslatorConfig{DefaultLanguage: n.systemDefaults.DefaultLanguage})
	if err != nil {
		return nil, err
	}
	ctx := context.TODO()
	allCustomTexts, err := n.queries.CustomTextListByTemplate(ctx, domain.IAMID, textType)
	if err != nil {
		return translator, nil
	}
	customTexts, err := n.queries.CustomTextListByTemplate(ctx, orgID, textType)
	if err != nil {
		return translator, nil
	}
	allCustomTexts.CustomTexts = append(allCustomTexts.CustomTexts, customTexts.CustomTexts...)

	for _, text := range allCustomTexts.CustomTexts {
		msg := i18n.Message{
			ID:   text.Template + "." + text.Key,
			Text: text.Text,
		}
		translator.AddMessages(text.Language, msg)
	}
	return translator, nil
}

func (n *Notification) getUserByID(userID string) (*model.NotifyUser, error) {
	return n.view.NotifyUserByID(userID)
}
