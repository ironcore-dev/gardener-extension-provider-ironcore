// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ignition

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/netip"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	buconfig "github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	"github.com/imdario/mergo"
	"sigs.k8s.io/yaml"
)

var (
	//go:embed template.yaml
	IgnitionTemplate string
)

const (
	dnsConfFile    = "/etc/systemd/resolved.conf.d/dns.conf"
	dnsEqualString = "DNS="
	fileMode       = 0644
)

type Config struct {
	Hostname   string
	UserData   string
	DnsServers []netip.Addr
}

func File(config *Config) (string, error) {
	ignitionBase := &map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(IgnitionTemplate), ignitionBase); err != nil {
		return "", err
	}

	if len(config.DnsServers) > 0 {
		dnsServers := []string{"[Resolve]"}
		for _, v := range config.DnsServers {
			dnsEntry := fmt.Sprintf("%s%s", dnsEqualString, v.String())
			dnsServers = append(dnsServers, dnsEntry)
		}

		dnsConf := map[string]interface{}{
			"storage": map[string]interface{}{
				"files": []interface{}{map[string]interface{}{
					"path": dnsConfFile,
					"mode": fileMode,
					"contents": map[string]interface{}{
						"inline": strings.Join(dnsServers, "\n"),
					},
				}},
			},
		}

		// merge dnsConfiguration with ignition content
		if err := mergo.Merge(ignitionBase, dnsConf, mergo.WithAppendSlice); err != nil {
			return "", fmt.Errorf("failed to merge dnsServer configuration with igntition content: %w", err)
		}
	}

	mergedIgnition, err := yaml.Marshal(ignitionBase)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("ignition").Funcs(sprig.HermeticTxtFuncMap()).Parse(string(mergedIgnition))
	if err != nil {
		return "", fmt.Errorf("failed creating ignition file: %w", err)
	}
	buf := bytes.NewBufferString("")
	err = tmpl.Execute(buf, config)
	if err != nil {
		return "", fmt.Errorf("failed creating ignition file while executing template: %w", err)
	}

	ignition, err := renderButane(buf.Bytes())
	if err != nil {
		return "", err
	}

	return ignition, nil
}
func renderButane(dataIn []byte) (string, error) {
	// render by butane to json
	options := common.TranslateBytesOptions{
		Raw:    true,
		Pretty: false,
	}
	options.NoResourceAutoCompression = true
	dataOut, _, err := buconfig.TranslateBytes(dataIn, options)
	if err != nil {
		return "", err
	}
	return string(dataOut), nil
}
