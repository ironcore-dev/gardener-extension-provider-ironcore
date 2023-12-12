// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

//go:generate sh -c "bash $GARDENER_HACK_DIR/generate-controller-registration.sh provider-ironcore . $(cat ../../VERSION) ../../example/controller-registration.yaml BackupBucket:ironcore BackupEntry:ironcore Bastion:ironcore ControlPlane:ironcore Infrastructure:ironcore Worker:ironcore"

// Package chart enables go:generate support for generating the correct controller registration.
package chart
