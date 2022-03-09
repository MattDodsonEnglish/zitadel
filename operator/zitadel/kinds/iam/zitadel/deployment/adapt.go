package deployment

import (
	"fmt"
	"time"

	"github.com/caos/zitadel/pkg/databases/db"

	"github.com/caos/orbos/mntr"
	"github.com/caos/orbos/pkg/kubernetes"
	"github.com/caos/orbos/pkg/kubernetes/k8s"
	"github.com/caos/orbos/pkg/kubernetes/resources/deployment"
	"github.com/caos/orbos/pkg/labels"
	"github.com/caos/zitadel/operator"
	"github.com/caos/zitadel/operator/helpers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dbSecrets     = "db-secrets"
	containerName = "zitadel"
	RunAsUser     = int64(1000)
	//zitadelImage can be found in github.com/caos/zitadel repo
	timeout = 60 * time.Second
)

func AdaptFunc(
	monitor mntr.Monitor,
	nameLabels *labels.Name,
	podSelector *labels.Selector,
	force bool,
	version *string,
	namespace string,
	replicaCount int,
	affinity *k8s.Affinity,
	cmName string,
	certPath string,
	secretName string,
	secretPath string,
	consoleCMName string,
	secretVarsName string,
	secretPasswordsName string,
	nodeSelector map[string]string,
	tolerations []corev1.Toleration,
	resources *k8s.Resources,
	migrationDone operator.EnsureFunc,
	configurationDone operator.EnsureFunc,
	setupDone operator.EnsureFunc,
	customImageRegistry string,
	dbConn db.Connection,
) (
	func(
		getConfigurationHashes func(k8sClient kubernetes.ClientInt, queried map[string]interface{}) (map[string]string, error),
	) operator.QueryFunc,
	operator.DestroyFunc,
	error,
) {
	internalMonitor := monitor.WithField("type", "deployment")

	destroy, err := deployment.AdaptFuncToDestroy(namespace, nameLabels.Name())
	if err != nil {
		return nil, nil, err
	}
	destroyers := []operator.DestroyFunc{
		operator.ResourceDestroyToZitadelDestroy(destroy),
	}

	return func(
			getConfigurationHashes func(k8sClient kubernetes.ClientInt, queried map[string]interface{}) (map[string]string, error),
		) operator.QueryFunc {
			return func(k8sClient kubernetes.ClientInt, queried map[string]interface{}) (operator.EnsureFunc, error) {
				deploymentDef := deploymentDef(
					nameLabels,
					namespace,
					replicaCount,
					podSelector,
					nodeSelector,
					tolerations,
					affinity,
					version,
					resources,
					cmName,
					certPath,
					secretName,
					secretPath,
					consoleCMName,
					secretVarsName,
					secretPasswordsName,
					customImageRegistry,
					dbConn,
				)

				hashes, err := getConfigurationHashes(k8sClient, queried)
				if err != nil {
					return nil, err
				}
				if hashes != nil && len(hashes) != 0 {
					for k, v := range hashes {
						deploymentDef.Annotations[k] = v
						deploymentDef.Spec.Template.Annotations[k] = v
					}
				}

				query, err := deployment.AdaptFuncToEnsure(deploymentDef, force)
				if err != nil {
					return nil, err
				}

				queriers := []operator.QueryFunc{
					operator.EnsureFuncToQueryFunc(migrationDone),
					operator.EnsureFuncToQueryFunc(configurationDone),
					operator.EnsureFuncToQueryFunc(setupDone),
					operator.ResourceQueryToZitadelQuery(query),
				}

				return operator.QueriersToEnsureFunc(internalMonitor, false, queriers, k8sClient, queried)
			}
		},
		operator.DestroyersToDestroyFunc(internalMonitor, destroyers),
		nil

}

func deploymentDef(
	nameLabels *labels.Name,
	namespace string,
	replicaCount int,
	podSelector *labels.Selector,
	nodeSelector map[string]string,
	tolerations []corev1.Toleration,
	affinity *k8s.Affinity,
	version *string,
	resources *k8s.Resources,
	cmName string,
	certPath string,
	secretName string,
	secretPath string,
	consoleCMName string,
	secretVarsName string,
	secretPasswordsName string,
	customImageRegistry string,
	dbConn db.Connection,
) *appsv1.Deployment {

	chownedVolumeMount := corev1.VolumeMount{
		Name:      "chowned-certs",
		MountPath: certPath,
	}

	srcVolume, destVolume, chownCertsContainer := db.InitChownCerts(customImageRegistry, fmt.Sprintf("%d:%d", RunAsUser, RunAsUser), corev1.VolumeMount{
		Name:      "certs",
		MountPath: "certificates",
	}, chownedVolumeMount)

	deploymentDef := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nameLabels.Name(),
			Namespace:   namespace,
			Labels:      labels.MustK8sMap(nameLabels),
			Annotations: map[string]string{},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: helpers.PointerInt32(int32(replicaCount)),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.MustK8sMap(podSelector),
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: helpers.IntToIntStr(1),
					MaxSurge:       helpers.IntToIntStr(1),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels.MustK8sMap(labels.AsSelectable(nameLabels)),
					Annotations: map[string]string{},
				},
				Spec: corev1.PodSpec{
					NodeSelector:   nodeSelector,
					Tolerations:    tolerations,
					Affinity:       affinity.K8s(),
					InitContainers: []corev1.Container{chownCertsContainer},
					Containers: []corev1.Container{
						GetContainer(
							containerName,
							*version,
							RunAsUser,
							true,
							GetResourcesFromDefault(resources),
							cmName,
							secretName,
							secretPath,
							consoleCMName,
							secretVarsName,
							chownedVolumeMount,
							"start",
							customImageRegistry,
							dbConn,
						),
					},
					Volumes: append(GetVolumes(
						secretName,
						secretPasswordsName,
						consoleCMName,
					), srcVolume, destVolume),
				},
			},
		},
	}
	return deploymentDef
}
