// Copyright 2022 IronCore authors
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

package admission

import (
	"github.com/gardener/gardener/extensions/pkg/util"
	ironcore "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
	"k8s.io/apimachinery/pkg/runtime"
)

// DecodeControlPlaneConfig decodes the `ControlPlaneConfig` from the given `RawExtension`.
func DecodeControlPlaneConfig(decoder runtime.Decoder, cp *runtime.RawExtension) (*ironcore.ControlPlaneConfig, error) {
	controlPlaneConfig := &ironcore.ControlPlaneConfig{}
	if err := util.Decode(decoder, cp.Raw, controlPlaneConfig); err != nil {
		return nil, err
	}

	return controlPlaneConfig, nil
}

// DecodeInfrastructureConfig decodes the `InfrastructureConfig` from the given `RawExtension`.
func DecodeInfrastructureConfig(decoder runtime.Decoder, infra *runtime.RawExtension) (*ironcore.InfrastructureConfig, error) {
	infraConfig := &ironcore.InfrastructureConfig{}
	if err := util.Decode(decoder, infra.Raw, infraConfig); err != nil {
		return nil, err
	}

	return infraConfig, nil
}
