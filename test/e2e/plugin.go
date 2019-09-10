// Copyright 2019 The knative Authors

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"testing"
)

type plugin struct {
	t *testing.T
	l Logger
}

// Create kn plugin folder if need
func (p plugin) CreatePluginFolder() (string, error) {
	return runCLIWithOpts("mkdir", []string{"-p", KN_PLUGIN_FOLDER}, runOpts{}, p.l)
}

// Clean the tkn cli folder
func (p plugin) CleanTknCli() (string, error) {
	return runCLIWithOpts("rm", []string{"-rf", KN_PLUGIN_FOLDER + "/" + TKN_CLI_FOLDER}, runOpts{}, p.l)
}

// Copy the tkn plugin to the kn plugin fplder
func (p plugin) CopyPlugin(name string, args []string, opts runOpts) (string, error) {
	path := KN_PLUGIN_PATH + "/" + name + "/kn-" + name
	return runCLIWithOpts("cp", []string{path, KN_PLUGIN_FOLDER}, opts, p.l)
}

// Delete the tkn plugin
func (p plugin) DeletePlugin(name string, args []string, opts runOpts) (string, error) {
	return runCLIWithOpts("rm", []string{"-rf", KN_PLUGIN_FOLDER + "/kn-" + name}, opts, p.l)
}
