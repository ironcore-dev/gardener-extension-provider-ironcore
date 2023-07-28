// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	extensionsbackupbucketcontroller "github.com/gardener/gardener/extensions/pkg/controller/backupbucket"
	extensionsbackupentrycontroller "github.com/gardener/gardener/extensions/pkg/controller/backupentry"
	extensionsbastioncontroller "github.com/gardener/gardener/extensions/pkg/controller/bastion"
	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	extensionscontrolplanecontroller "github.com/gardener/gardener/extensions/pkg/controller/controlplane"
	extensionshealthcheckcontroller "github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	extensionsheartbeatcontroller "github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	extensionsinfrastructurecontroller "github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	extensionsworkercontroller "github.com/gardener/gardener/extensions/pkg/controller/worker"
	extensionscloudproviderwebhook "github.com/gardener/gardener/extensions/pkg/webhook/cloudprovider"
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"
	extensioncontrolplanewebhook "github.com/gardener/gardener/extensions/pkg/webhook/controlplane"
	backupbucketcontroller "github.com/onmetal/gardener-extension-provider-onmetal/pkg/controller/backupbucket"
	backupentrycontroller "github.com/onmetal/gardener-extension-provider-onmetal/pkg/controller/backupentry"
	bastioncontroller "github.com/onmetal/gardener-extension-provider-onmetal/pkg/controller/bastion"
	controlplanecontroller "github.com/onmetal/gardener-extension-provider-onmetal/pkg/controller/controlplane"
	healthcheckcontroller "github.com/onmetal/gardener-extension-provider-onmetal/pkg/controller/healthcheck"
	infrastructurecontroller "github.com/onmetal/gardener-extension-provider-onmetal/pkg/controller/infrastructure"
	workercontroller "github.com/onmetal/gardener-extension-provider-onmetal/pkg/controller/worker"
	cloudproviderwebhook "github.com/onmetal/gardener-extension-provider-onmetal/pkg/webhook/cloudprovider"
	controlplanewebhook "github.com/onmetal/gardener-extension-provider-onmetal/pkg/webhook/controlplane"
)

// ControllerSwitchOptions are the controllercmd.SwitchOptions for the provider controllers.
func ControllerSwitchOptions() *controllercmd.SwitchOptions {
	return controllercmd.NewSwitchOptions(
		controllercmd.Switch(extensionsbackupbucketcontroller.ControllerName, backupbucketcontroller.AddToManager),
		controllercmd.Switch(extensionsbackupentrycontroller.ControllerName, backupentrycontroller.AddToManager),
		controllercmd.Switch(extensionsbastioncontroller.ControllerName, bastioncontroller.AddToManager),
		controllercmd.Switch(extensionscontrolplanecontroller.ControllerName, controlplanecontroller.AddToManager),
		controllercmd.Switch(extensionsinfrastructurecontroller.ControllerName, infrastructurecontroller.AddToManager),
		controllercmd.Switch(extensionsworkercontroller.ControllerName, workercontroller.AddToManager),
		controllercmd.Switch(extensionshealthcheckcontroller.ControllerName, healthcheckcontroller.AddToManager),
		controllercmd.Switch(extensionsheartbeatcontroller.ControllerName, extensionsheartbeatcontroller.AddToManager),
	)
}

// WebhookSwitchOptions are the webhookcmd.SwitchOptions for the provider webhooks.
func WebhookSwitchOptions() *webhookcmd.SwitchOptions {
	return webhookcmd.NewSwitchOptions(
		webhookcmd.Switch(extensioncontrolplanewebhook.WebhookName, controlplanewebhook.AddToManager),
		webhookcmd.Switch(extensionscloudproviderwebhook.WebhookName, cloudproviderwebhook.AddToManager),
	)
}
