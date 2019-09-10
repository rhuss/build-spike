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

func TestTaskRun(t *testing.T) {

	t.Parallel()
	test := NewE2eTest(t)
	test.Setup(t, PLUGIN_TASKRUN_NAME)
	defer test.Teardown(t, PLUGIN_TASKRUN_NAME)

	t.Run("shows tkn taskrun plugin command", func(t *testing.T) {
		test.taskrunPlugins(t)
	})

	t.Run("returns tkn taskrun main command", func(t *testing.T) {
		test.taskrunMain(t)
	})

	t.Run("returns tkn taskrun list command", func(t *testing.T) {
		test.taskrunList(t)
	})
}

func (test *e2eTest) taskrunPlugins(t *testing.T) {
	out, err := test.kn.RunWithOpts([]string{"plugin", "list"}, runOpts{NoNamespace: true})
	assert.NilError(t, err)
	assert.Check(t, ContainsAll(out, KN_PLUGIN_FOLDER+"/kn-"+PLUGIN_TASKRUN_NAME))
}

func (test *e2eTest) taskrunMain(t *testing.T) {
	out, err := test.kn.RunWithOpts([]string{PLUGIN_TASKRUN_NAME}, runOpts{NoNamespace: true})
	assert.NilError(t, err)
	assert.Check(t, ContainsAll(out, "list"))
}

func (test *e2eTest) taskrunList(t *testing.T) {
	out, err := test.kn.RunWithOpts([]string{PLUGIN_TASKRUN_NAME, "list"}, runOpts{NoNamespace: true})
	assert.NilError(t, err)
	assert.Check(t, ContainsAll(out, "NAME"))
}
