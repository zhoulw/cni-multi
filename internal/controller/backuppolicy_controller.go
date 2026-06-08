/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	multiappv1alpha1 "cni-multi/api/v1alpha1"
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BackupPolicyReconciler reconciles a BackupPolicy object
type BackupPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=multiapp.zlw.domain,resources=backuppolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=multiapp.zlw.domain,resources=backuppolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=multiapp.zlw.domain,resources=backuppolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BackupPolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *BackupPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// TODO(user): your logic here
	// 获取backuppolicy
	var policy multiappv1alpha1.BackupPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. 构建期望的 CronJob
	desired := r.buildCronJob(&policy)

	// 3. 设置 OwnerReference，使 CronJob 随 BackupPolicy 自动删除
	if err := controllerutil.SetControllerReference(&policy, desired, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// 4. 获取同名的现存 CronJob
	var cronJob batchv1.CronJob
	err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: desired.Name}, &cronJob)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if err != nil {
		// 不存在 -> 创建
		logger.Info("Creating CronJob", "name", desired.Name)
		if err := r.Create(ctx, desired); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// 已存在 -> 更新（仅当 spec 有变化时）
		if !cronJobSpecEqual(&cronJob.Spec, &desired.Spec) {
			cronJob.Spec = desired.Spec
			logger.Info("Updating CronJob", "name", desired.Name)
			if err := r.Update(ctx, &cronJob); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// 5. 更新 status
	policy.Status.ActiveCronJob = desired.Name
	if err := r.Status().Update(ctx, &policy); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BackupPolicyReconciler) buildCronJob(policy *multiappv1alpha1.BackupPolicy) *batchv1.CronJob {
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "backuppolicy-controller",
		"backuppolicy":                 policy.Name,
	}

	image := policy.Spec.BackupImage
	if image == "" {
		image = "bitnami/etcd:latest"
	}

	// 备份并清理旧文件的命令
	backupCmd := fmt.Sprintf(
		`find /backup -type f -name '*.db' -mtime +%d -delete && \
         etcdctl --endpoints=%s \
                 --cacert=/etc/etcd-secrets/ca.crt \
                 --cert=/etc/etcd-secrets/client.crt \
                 --key=/etc/etcd-secrets/client.key \
                 snapshot save /backup/etcd-snapshot-$(date +%%Y%%m%%d%%H%%M%%S).db`,
		policy.Spec.RetentionDays,
		policy.Spec.EtcdEndpoint,
	)

	ttlSecondsAfterFinished := int32(3600) // 保留已完成的 Job 一小时方便查看

	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("backup-%s", policy.Name),
			Namespace: policy.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          policy.Spec.Schedule,
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							HostNetwork:   true,
							DNSPolicy:     corev1.DNSClusterFirstWithHostNet,
							RestartPolicy: corev1.RestartPolicyOnFailure,
							NodeSelector:  policy.Spec.NodeSelector,
							Tolerations:   policy.Spec.Tolerations,
							Containers: []corev1.Container{
								{
									Name:    "backup",
									Image:   image,
									Command: []string{"/bin/sh", "-c"},
									Args:    []string{backupCmd},
									VolumeMounts: []corev1.VolumeMount{
										{Name: "backup-storage", MountPath: "/backup"},
										{Name: "etcd-secrets", MountPath: "/etc/etcd-secrets", ReadOnly: true},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "backup-storage",
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: policy.Spec.StoragePVC,
										},
									},
								},
								{
									Name: "etcd-secrets",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: policy.Spec.EtcdSecretRef.Name,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func cronJobSpecEqual(a, b *batchv1.CronJobSpec) bool {
	// 简单比较，生产环境可引入更完善的 reflect 或 equality 库
	if a.Schedule != b.Schedule {
		return false
	}
	if a.ConcurrencyPolicy != b.ConcurrencyPolicy {
		return false
	}
	// 这里可以根据需要增加 JobTemplate 关键字段的比较
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multiappv1alpha1.BackupPolicy{}).
		Named("backuppolicy").
		Complete(r)
}
