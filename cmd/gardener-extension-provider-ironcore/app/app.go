// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	druidcorev1alpha1 "github.com/gardener/etcd-druid/api/core/v1alpha1"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	heartbeatcmd "github.com/gardener/gardener/extensions/pkg/controller/heartbeat/cmd"
	"github.com/gardener/gardener/extensions/pkg/util"
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	gardenerhealthz "github.com/gardener/gardener/pkg/healthz"
	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/component-base/version/verflag"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	ironcoreinstall "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/install"
	ironcorecmd "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/cmd"
	backupbucketcontroller "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/controller/backupbucket"
	backupentrycontroller "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/controller/backupentry"
	bastioncontroller "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/controller/bastion"
	ironcorecontrolplane "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/controller/controlplane"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/controller/healthcheck"
	infrastructurecontroller "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/controller/infrastructure"
	workercontroller "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/controller/worker"
	ironcore "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

// NewControllerManagerCommand creates a new command for running a ironcore provider controller.
func NewControllerManagerCommand(ctx context.Context) *cobra.Command {
	var (
		generalOpts = &controllercmd.GeneralOptions{}
		restOpts    = &controllercmd.RESTOptions{}
		mgrOpts     = &controllercmd.ManagerOptions{
			LeaderElection:          true,
			LeaderElectionID:        controllercmd.LeaderElectionNameID(ironcore.ProviderName),
			LeaderElectionNamespace: os.Getenv("LEADER_ELECTION_NAMESPACE"),
			WebhookServerPort:       443,
			WebhookCertDir:          "/tmp/gardener-extensions-cert",
			MetricsBindAddress:      ":8080",
			HealthBindAddress:       ":8081",
		}
		configFileOpts = &ironcorecmd.ConfigOptions{}

		// options for the backupbucket controller
		backupBucketCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the backupentry controller
		backupEntryCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the health care controller
		healthCheckCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the heartbeat controller
		heartbeatCtrlOpts = &heartbeatcmd.Options{
			ExtensionName:        ironcore.ProviderName,
			RenewIntervalSeconds: 30,
			Namespace:            os.Getenv("LEADER_ELECTION_NAMESPACE"),
		}

		// options for the controlplane controller
		controlPlaneCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the infrastructure controller
		infraCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		reconcileOpts = &controllercmd.ReconcilerOptions{}

		// options for the worker controller
		workerCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the webhook server
		webhookServerOptions = &webhookcmd.ServerOptions{
			Namespace: os.Getenv("WEBHOOK_CONFIG_NAMESPACE"),
		}

		// options for the bastion controller
		bastionCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		controllerSwitches = ironcorecmd.ControllerSwitchOptions()
		webhookSwitches    = ironcorecmd.WebhookSwitchOptions()
		webhookOptions     = webhookcmd.NewAddToManagerOptions(
			ironcore.ProviderName,
			genericactuator.ShootWebhooksResourceName,
			genericactuator.ShootWebhookNamespaceSelector(ironcore.Type),
			webhookServerOptions,
			webhookSwitches,
		)

		aggOption = controllercmd.NewOptionAggregator(
			generalOpts,
			restOpts,
			mgrOpts,
			controllercmd.PrefixOption("controlplane-", controlPlaneCtrlOpts),
			controllercmd.PrefixOption("infrastructure-", infraCtrlOpts),
			controllercmd.PrefixOption("worker-", workerCtrlOpts),
			controllercmd.PrefixOption("healthcheck-", healthCheckCtrlOpts),
			controllercmd.PrefixOption("heartbeat-", heartbeatCtrlOpts),
			controllercmd.PrefixOption("bastion-", bastionCtrlOpts),
			controllercmd.PrefixOption("backupbucket-", backupBucketCtrlOpts),
			controllercmd.PrefixOption("backupentry-", backupEntryCtrlOpts),
			configFileOpts,
			controllerSwitches,
			reconcileOpts,
			webhookOptions,
		)
	)

	cmd := &cobra.Command{
		Use: fmt.Sprintf("%s-controller-manager", ironcore.ProviderName),

		RunE: func(cmd *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()

			if err := aggOption.Complete(); err != nil {
				return fmt.Errorf("error completing options: %w", err)
			}

			if err := heartbeatCtrlOpts.Validate(); err != nil {
				return err
			}

			util.ApplyClientConnectionConfigurationToRESTConfig(configFileOpts.Completed().Config.ClientConnection, restOpts.Completed().Config)

			mopts := mgrOpts.Completed().Options()
			mopts.Client = client.Options{
				Cache: &client.CacheOptions{
					DisableFor: []client.Object{
						&corev1.Secret{},
					},
				},
			}
			mgr, err := manager.New(restOpts.Completed().Config, mopts)
			if err != nil {
				return fmt.Errorf("could not instantiate manager: %w", err)
			}

			scheme := mgr.GetScheme()
			if err := extensionscontroller.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := ironcoreinstall.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := druidcorev1alpha1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := autoscalingv1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := machinev1alpha1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}

			// add common meta types to schema for controller-runtime to use v1.ListOptions
			metav1.AddToGroupVersion(scheme, machinev1alpha1.SchemeGroupVersion)

			log := mgr.GetLogger()
			log.Info("Getting rest config for garden")
			gardenRESTConfig, err := kubernetes.RESTConfigFromKubeconfigFile(os.Getenv("GARDEN_KUBECONFIG"), kubernetes.AuthTokenFile)
			if err != nil {
				return err
			}

			log.Info("Setting up cluster object for garden")
			gardenCluster, err := cluster.New(gardenRESTConfig, func(opts *cluster.Options) {
				opts.Scheme = kubernetes.GardenScheme
				opts.Logger = log
			})
			if err != nil {
				return fmt.Errorf("failed creating garden cluster object: %w", err)
			}

			log.Info("Adding garden cluster to manager")
			if err := mgr.Add(gardenCluster); err != nil {
				return fmt.Errorf("failed adding garden cluster to manager: %w", err)
			}

			configFileOpts.Completed().ApplyHealthCheckConfig(&healthcheck.DefaultAddOptions.HealthCheckConfig)
			healthCheckCtrlOpts.Completed().Apply(&healthcheck.DefaultAddOptions.Controller)
			configFileOpts.Completed().ApplyBastionConfig(&bastioncontroller.DefaultAddOptions.BastionConfig)
			heartbeatCtrlOpts.Completed().Apply(&heartbeat.DefaultAddOptions)
			infraCtrlOpts.Completed().Apply(&infrastructurecontroller.DefaultAddOptions.Controller)
			workerCtrlOpts.Completed().Apply(&workercontroller.DefaultAddOptions.Controller)
			configFileOpts.Completed().ApplyBackupbucketConfig(&backupbucketcontroller.DefaultAddOptions.BackupBucketConfig)
			bastionCtrlOpts.Completed().Apply(&bastioncontroller.DefaultAddOptions.Controller)
			backupBucketCtrlOpts.Completed().Apply(&backupbucketcontroller.DefaultAddOptions.Controller)
			backupEntryCtrlOpts.Completed().Apply(&backupentrycontroller.DefaultAddOptions.Controller)
			reconcileOpts.Completed().Apply(&bastioncontroller.DefaultAddOptions.IgnoreOperationAnnotation, &bastioncontroller.DefaultAddOptions.ExtensionClass)
			reconcileOpts.Completed().Apply(&infrastructurecontroller.DefaultAddOptions.IgnoreOperationAnnotation, &infrastructurecontroller.DefaultAddOptions.ExtensionClass)
			reconcileOpts.Completed().Apply(&workercontroller.DefaultAddOptions.IgnoreOperationAnnotation, &workercontroller.DefaultAddOptions.ExtensionClass)
			reconcileOpts.Completed().Apply(&backupbucketcontroller.DefaultAddOptions.IgnoreOperationAnnotation, &backupbucketcontroller.DefaultAddOptions.ExtensionClass)
			reconcileOpts.Completed().Apply(&backupentrycontroller.DefaultAddOptions.IgnoreOperationAnnotation, &backupentrycontroller.DefaultAddOptions.ExtensionClass)
			workercontroller.DefaultAddOptions.GardenCluster = gardenCluster
			workercontroller.DefaultAddOptions.AutonomousShootCluster = generalOpts.Completed().AutonomousShootCluster

			if _, err := webhookOptions.Completed().AddToManager(ctx, mgr, nil, generalOpts.Completed().AutonomousShootCluster); err != nil {
				return fmt.Errorf("could not add webhooks to manager: %w", err)
			}
			ironcorecontrolplane.DefaultAddOptions.WebhookServerNamespace = webhookOptions.Server.Namespace

			if err := controllerSwitches.Completed().AddToManager(ctx, mgr); err != nil {
				return fmt.Errorf("could not add controllers to manager: %w", err)
			}

			if err := mgr.AddReadyzCheck("informer-sync", gardenerhealthz.NewCacheSyncHealthz(mgr.GetCache())); err != nil {
				return fmt.Errorf("could not add readycheck for informers: %w", err)
			}

			if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
				return fmt.Errorf("could not add health check to manager: %w", err)
			}

			if err := mgr.AddReadyzCheck("webhook-server", mgr.GetWebhookServer().StartedChecker()); err != nil {
				return fmt.Errorf("could not add ready check for webhook server to manager: %w", err)
			}

			if err := mgr.Start(ctx); err != nil {
				return fmt.Errorf("error running manager: %w", err)
			}

			return nil
		},
	}

	verflag.AddFlags(cmd.Flags())
	aggOption.AddFlags(cmd.Flags())

	return cmd
}
