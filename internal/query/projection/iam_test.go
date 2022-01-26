package projection

import (
	"testing"

	"github.com/caos/zitadel/internal/domain"
	"github.com/caos/zitadel/internal/errors"
	"github.com/caos/zitadel/internal/eventstore"
	"github.com/caos/zitadel/internal/eventstore/handler"
	"github.com/caos/zitadel/internal/eventstore/repository"
	"github.com/caos/zitadel/internal/repository/iam"
)

func TestIAMProjection_reduces(t *testing.T) {
	type args struct {
		event func(t *testing.T) eventstore.Event
	}
	tests := []struct {
		name   string
		args   args
		reduce func(event eventstore.Event) (*handler.Statement, error)
		want   wantReduce
	}{
		{
			name: "reduceGlobalOrgSet",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(iam.GlobalOrgSetEventType),
					iam.AggregateType,
					[]byte(`{"globalOrgId": "orgid"}`),
				), iam.GlobalOrgSetMapper),
			},
			reduce: (&IAMProjection{}).reduceGlobalOrgSet,
			want: wantReduce{
				projection:       IAMProjectionTable,
				aggregateType:    eventstore.AggregateType("iam"),
				sequence:         15,
				previousSequence: 10,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPSERT INTO zitadel.projections.iam (id, change_date, sequence, global_org_id) VALUES ($1, $2, $3, $4)",
							expectedArgs: []interface{}{
								"agg-id",
								anyArg{},
								uint64(15),
								"orgid",
							},
						},
					},
				},
			},
		},
		{
			name: "reduceProjectIDSet",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(iam.ProjectSetEventType),
					iam.AggregateType,
					[]byte(`{"iamProjectId": "project-id"}`),
				), iam.ProjectSetMapper),
			},
			reduce: (&IAMProjection{}).reduceIAMProjectSet,
			want: wantReduce{
				projection:       IAMProjectionTable,
				aggregateType:    eventstore.AggregateType("iam"),
				sequence:         15,
				previousSequence: 10,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPSERT INTO zitadel.projections.iam (id, change_date, sequence, iam_project_id) VALUES ($1, $2, $3, $4)",
							expectedArgs: []interface{}{
								"agg-id",
								anyArg{},
								uint64(15),
								"project-id",
							},
						},
					},
				},
			},
		},
		{
			name: "reduceDefaultLanguageSet",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(iam.DefaultLanguageSetEventType),
					iam.AggregateType,
					[]byte(`{"defaultLanguage": "en"}`),
				), iam.DefaultLanguageSetMapper),
			},
			reduce: (&IAMProjection{}).reduceDefaultLanguageSet,
			want: wantReduce{
				projection:       IAMProjectionTable,
				aggregateType:    eventstore.AggregateType("iam"),
				sequence:         15,
				previousSequence: 10,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPSERT INTO zitadel.projections.iam (id, change_date, sequence, default_language) VALUES ($1, $2, $3, $4)",
							expectedArgs: []interface{}{
								"agg-id",
								anyArg{},
								uint64(15),
								"en",
							},
						},
					},
				},
			},
		},
		{
			name: "reduceSetupStarted",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(iam.SetupStartedEventType),
					iam.AggregateType,
					[]byte(`{"Step": 1}`),
				), iam.SetupStepMapper),
			},
			reduce: (&IAMProjection{}).reduceSetupEvent,
			want: wantReduce{
				projection:       IAMProjectionTable,
				aggregateType:    eventstore.AggregateType("iam"),
				sequence:         15,
				previousSequence: 10,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPSERT INTO zitadel.projections.iam (id, change_date, sequence, setup_started) VALUES ($1, $2, $3, $4)",
							expectedArgs: []interface{}{
								"agg-id",
								anyArg{},
								uint64(15),
								domain.Step1,
							},
						},
					},
				},
			},
		},
		{
			name: "reduceSetupDone",
			args: args{
				event: getEvent(testEvent(
					repository.EventType(iam.SetupDoneEventType),
					iam.AggregateType,
					[]byte(`{"Step": 1}`),
				), iam.SetupStepMapper),
			},
			reduce: (&IAMProjection{}).reduceSetupEvent,
			want: wantReduce{
				projection:       IAMProjectionTable,
				aggregateType:    eventstore.AggregateType("iam"),
				sequence:         15,
				previousSequence: 10,
				executer: &testExecuter{
					executions: []execution{
						{
							expectedStmt: "UPSERT INTO zitadel.projections.iam (id, change_date, sequence, setup_done) VALUES ($1, $2, $3, $4)",
							expectedArgs: []interface{}{
								"agg-id",
								anyArg{},
								uint64(15),
								domain.Step1,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := baseEvent(t)
			got, err := tt.reduce(event)
			if _, ok := err.(errors.InvalidArgument); !ok {
				t.Errorf("no wrong event mapping: %v, got: %v", err, got)
			}

			event = tt.args.event(t)
			got, err = tt.reduce(event)
			assertReduce(t, got, err, tt.want)
		})
	}
}
