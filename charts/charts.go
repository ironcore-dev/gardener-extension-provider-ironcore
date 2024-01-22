// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package charts

import (
	"embed"
)

// InternalChart embeds the internal charts in embed.FS
//
//go:embed internal
var InternalChart embed.FS

const InternalChartsPath = "internal"
