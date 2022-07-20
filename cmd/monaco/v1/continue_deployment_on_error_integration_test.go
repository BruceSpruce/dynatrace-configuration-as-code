//go:build integration_v1
// +build integration_v1

/**
 * @license
 * Copyright 2020 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/v2/runner"
	"path/filepath"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/spf13/afero"

	"gotest.tools/assert"
)

// tests all configs for a single environment
func TestIntegrationContinueDeploymentOnError(t *testing.T) {

	allConfigsFolder := AbsOrPanicFromSlash("test-resources/integration-configs-with-errors/")
	allConfigsEnvironmentsFile := filepath.Join(allConfigsFolder, "environments.yaml")

	RunLegacyIntegrationWithCleanup(t, allConfigsFolder, allConfigsEnvironmentsFile, "AllConfigs", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", allConfigsEnvironmentsFile, fs)
		assert.Check(t, len(errs) == 0, "didn't expect errors loading test environments")

		projects, err := project.LoadProjectsToDeploy(fs, "", api.NewApis(), allConfigsFolder)
		assert.NilError(t, err)

		statusCode := runner.RunImpl([]string{
			"monaco",
			"deploy",
			"--verbose",
			"--continue-on-error",
			"--environments", allConfigsEnvironmentsFile,
			allConfigsFolder,
		}, fs)

		// deployment should fail
		assert.Equal(t, statusCode, 1)

		// dashboard should anyways be deployed
		dashboardConfig, err := projects[0].GetConfig("dashboard")
		assert.NilError(t, err)
		AssertConfigAvailability(t, dashboardConfig, environments["environment1"], true)
	})
}
