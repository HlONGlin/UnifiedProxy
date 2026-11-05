// Package scripts embeds the public node bootstrap shipped with the control.
package scripts

import _ "embed"

//go:embed install-node.sh
var NodeInstaller string
