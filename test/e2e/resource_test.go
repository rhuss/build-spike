// Copyright 2019 The Knative Authors

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build e2e

package e2e

import (
	"gotest.tools/assert"
	"testing"
)

func TestResource(t *testing.T) {

	t.Parallel()
	test := NewE2eTest(t)
	test.Setup(t, PLUGIN_RESOURCE_NAME)
	defer test.Teardown(t, PLUGIN_RESOURCE_NAME)

	t.Run("shows tkn resource plugin command", func(t *testing.T) {
		test.resourcePlugins(t)
	})

	t.Run("returns tkn resource main command", func(t *testing.T) {
		test.resourceMain(t)
	})

	t.Run("returns tkn resource list command", func(t *testing.T) {
		test.resourceList(t)
	})
}

func (test *e2eTest) resourcePlugins(t *testing.T) {
	out, err := test.kn.RunWithOpts([]string{"plugin", "list"}, runOpts{NoNamespace: true})
	assert.NilError(t, err)
	assert.Check(t, ContainsAll(out, KN_PLUGIN_FOLDER+"/kn-"+PLUGIN_RESOURCE_NAME))
}

func (test *e2eTest) resourceMain(t *testing.T) {
	out, err := test.kn.RunWithOpts([]string{PLUGIN_RESOURCE_NAME}, runOpts{NoNamespace: true})
	assert.NilError(t, err)
	assert.Check(t, ContainsAll(out, "list"))
}

func (test *e2eTest) resourceList(t *testing.T) {
	out, err := test.kn.RunWithOpts([]string{PLUGIN_RESOURCE_NAME, "list"}, runOpts{NoNamespace: true})
	assert.NilError(t, err)
	assert.Check(t, ContainsAll(out, "NAME"))
}
