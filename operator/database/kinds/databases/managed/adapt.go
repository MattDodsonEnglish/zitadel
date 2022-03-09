package managed

import (
	"fmt"
	"github.com/caos/zitadel/operator/database/kinds/databases/managed/user"
	"strconv"
	"strings"

	"github.com/caos/zitadel/pkg/databases/db"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/caos/orbos/mntr"
	"github.com/caos/orbos/pkg/kubernetes"
	"github.com/caos/orbos/pkg/kubernetes/resources/pdb"
	"github.com/caos/orbos/pkg/labels"
	"github.com/caos/orbos/pkg/secret"
	"github.com/caos/orbos/pkg/tree"

	"github.com/caos/zitadel/operator"
	"github.com/caos/zitadel/operator/common"
	"github.com/caos/zitadel/operator/database/kinds/backups"
	managedCurr "github.com/caos/zitadel/operator/database/kinds/databases/managed/current"
	"github.com/caos/zitadel/operator/database/kinds/databases/managed/rbac"
	"github.com/caos/zitadel/operator/database/kinds/databases/managed/services"
	"github.com/caos/zitadel/operator/database/kinds/databases/managed/statefulset"
	certCurr "github.com/caos/zitadel/operator/database/kinds/databases/managed/user/current"
)

const (
	SfsName            = "cockroachdb"
	pdbName            = SfsName + "-budget"
	serviceAccountName = SfsName
	PublicServiceName  = SfsName + "-public"
	privateServiceName = SfsName
	cockroachPort      = int32(26257)
	cockroachHTTPPort  = int32(8080)
	Clean              = "clean"
	DBReady            = "dbready"
	rootCertsSecret    = "root-certs"
	nodeCertsSecret    = "node-certs"
	zitadelCertsSecret = "zitadel-certs"
	clientCertsPath    = "/cockroach/cockroach-client-certs"
)

func Adapter(
	componentLabels *labels.Component,
	namespace string,
	timestamp string,
	nodeselector map[string]string,
	tolerations []corev1.Toleration,
	version string,
	features []string,
	customImageRegistry string,
) operator.AdaptFunc {

	return func(
		monitor mntr.Monitor,
		desired *tree.Tree,
		current *tree.Tree,
	) (
		_ operator.QueryFunc,
		_ operator.DestroyFunc,
		_ operator.ConfigureFunc,
		_ map[string]*secret.Secret,
		_ map[string]*secret.Existing,
		migrate bool,
		err error,
	) {

		defer func() {
			if err != nil {
				err = fmt.Errorf("adapting managed database failed: %w", err)
			}
		}()

		internalMonitor := monitor.WithField("kind", "cockroachdb")

		desiredKind, err := parseDesiredV0(desired)
		if err != nil {
			return nil, nil, nil, nil, nil, false, fmt.Errorf("parsing desired state failed: %w", err)
		}
		desired.Parsed = desiredKind

		storageCapacity, err := resource.ParseQuantity(desiredKind.Spec.StorageCapacity)
		if err != nil {
			return nil, nil, nil, nil, nil, false, mntr.ToUserError(fmt.Errorf("parsing storage capacity format failed: %w", err))
		}

		if !monitor.IsVerbose() && desiredKind.Spec.Verbose {
			internalMonitor.Verbose()
		}

		allSecrets, allExisting := getSecretsMap(desiredKind)

		var (
			isFeatureDatabase bool
			isFeatureClean    bool
		)
		for _, feature := range features {
			switch feature {
			case "database":
				isFeatureDatabase = true
			case Clean:
				isFeatureClean = true
			}
		}

		currentDB := &managedCurr.Current{
			Common: tree.NewCommon("databases.caos.ch/CockroachDB", "v0", false),
			Current: &managedCurr.CurrentDB{
				CA: &certCurr.Current{},
			},
		}
		current.Parsed = currentDB

		pwSecretLabels := labels.AsSelectable(labels.MustForName(componentLabels, "managed-db-password"))
		currentDB.Current.PasswordSecretKey = "zitadel"
		currentDB.Current.PasswordSecret = pwSecretLabels
		currentDB.Current.User = "zitadel"

		queryDBSetupBeforeCR, destroyDBSetupBeforeCR, queryDBSetupAfterCR, destroyDBSetupAfterCR, err := user.AdaptFunc(
			internalMonitor,
			namespace,
			componentLabels,
			desiredKind.Spec.ClusterDns,
			isFeatureDatabase,
			currentDB.Current.User,
			desiredKind.Spec.ZitadelUserPassword,
			desiredKind.Spec.ZitadelUserPasswordExisting,
			pwSecretLabels,
			currentDB.Current.PasswordSecretKey,
			rootCertsSecret,
			clientCertsPath,
			nodeCertsSecret,
		)
		if err != nil {
			return nil, nil, nil, nil, nil, false, err
		}

		queryRBAC, destroyRBAC, err := rbac.AdaptFunc(internalMonitor, namespace, labels.MustForName(componentLabels, serviceAccountName))
		if err != nil {
			return nil, nil, nil, nil, nil, false, err
		}

		cockroachNameLabels := labels.MustForName(componentLabels, SfsName)
		cockroachSelector := labels.DeriveNameSelector(cockroachNameLabels, false)
		cockroachSelectabel := labels.AsSelectable(cockroachNameLabels)
		querySFS, destroySFS, ensureInit, checkDBReady, err := statefulset.AdaptFunc(
			internalMonitor,
			cockroachSelectabel,
			cockroachSelector,
			desiredKind.Spec.Force,
			namespace,
			common.CockroachImage.Reference(customImageRegistry),
			serviceAccountName,
			desiredKind.Spec.ReplicaCount,
			storageCapacity,
			cockroachPort,
			cockroachHTTPPort,
			desiredKind.Spec.StorageClass,
			desiredKind.Spec.NodeSelector,
			desiredKind.Spec.Tolerations,
			desiredKind.Spec.Resources,
			desiredKind.Spec.Cache,
			desiredKind.Spec.MaxSQLMemory,
			clientCertsPath,
			rootCertsSecret,
			nodeCertsSecret,
		)
		if err != nil {
			return nil, nil, nil, nil, nil, false, err
		}

		queryS, destroyS, err := services.AdaptFunc(
			internalMonitor,
			namespace,
			labels.MustForName(componentLabels, PublicServiceName),
			labels.MustForName(componentLabels, privateServiceName),
			cockroachSelector,
			cockroachPort,
			cockroachHTTPPort,
		)

		queryPDB, err := pdb.AdaptFuncToEnsure(namespace, labels.MustForName(componentLabels, pdbName), cockroachSelector, "1")
		if err != nil {
			return nil, nil, nil, nil, nil, false, err
		}

		var (
			queriers    = make([]operator.QueryFunc, 0)
			destroyers  = make([]operator.DestroyFunc, 0)
			configurers = make([]operator.ConfigureFunc, 0)
		)
		if isFeatureDatabase {
			queriers = append(queriers,
				queryRBAC,
				queryDBSetupBeforeCR,
				operator.ResourceQueryToZitadelQuery(querySFS),
				queryS,
				operator.EnsureFuncToQueryFunc(ensureInit),
				queryDBSetupAfterCR,
				operator.ResourceQueryToZitadelQuery(queryPDB),
			)
			destroyers = append(destroyers,
				destroyS,
				destroyDBSetupAfterCR,
				operator.ResourceDestroyToZitadelDestroy(destroySFS),
				destroyDBSetupBeforeCR,
				destroyRBAC,
			)
		}
		if isFeatureClean {
			queriers = append(queriers,
				operator.ResourceQueryToZitadelQuery(
					statefulset.CleanPVCs(
						monitor,
						namespace,
						cockroachSelectabel,
						desiredKind.Spec.ReplicaCount,
					),
				),
				operator.EnsureFuncToQueryFunc(ensureInit),
				operator.EnsureFuncToQueryFunc(checkDBReady),
				queryDBSetupAfterCR,
			)
		}

		if desiredKind.Spec.Backups != nil {

			oneBackup := false
			for backupName := range desiredKind.Spec.Backups {
				if timestamp != "" && strings.HasPrefix(timestamp, backupName) {
					oneBackup = true
				}
			}

			for backupName, desiredBackup := range desiredKind.Spec.Backups {
				currentBackup := &tree.Tree{}
				if timestamp == "" || !oneBackup || (timestamp != "" && strings.HasPrefix(timestamp, backupName)) {
					queryB, destroyB, configureB, secrets, existing, migrateB, err := backups.Adapt(
						internalMonitor,
						desiredBackup,
						currentBackup,
						backupName,
						namespace,
						componentLabels,
						checkDBReady,
						strings.TrimPrefix(timestamp, backupName+"."),
						nodeselector,
						tolerations,
						version,
						currentDB,
						features,
						customImageRegistry,
					)
					if err != nil {
						return nil, nil, nil, nil, nil, false, err
					}

					migrate = migrate || migrateB

					secret.AppendSecrets(backupName, allSecrets, secrets, allExisting, existing)
					destroyers = append(destroyers, destroyB)
					queriers = append(queriers, queryB)
					configurers = append(configurers, configureB)
				}
			}
		}

		return func(k8sClient kubernetes.ClientInt, queried map[string]interface{}) (operator.EnsureFunc, error) {
				queriedCurrentDB, err := db.ParseQueriedForDatabase(queried)
				if err != nil || queriedCurrentDB == nil {
					// TODO: query system state
					currentDB.Current.Port = strconv.Itoa(int(cockroachPort))
					currentDB.Current.URL = PublicServiceName
					db.SetQueriedForDatabase(queried, current)
					internalMonitor.Info("set current state of managed database")
				}

				ensure, err := operator.QueriersToEnsureFunc(internalMonitor, true, queriers, k8sClient, queried)
				return ensure, err
			},
			operator.DestroyersToDestroyFunc(internalMonitor, destroyers),
			func(k8sClient kubernetes.ClientInt, queried map[string]interface{}, gitops bool) error {
				for i := range configurers {
					if err := configurers[i](k8sClient, queried, gitops); err != nil {
						return err
					}
				}
				return nil
			},
			allSecrets,
			allExisting,
			migrate,
			nil
	}
}
