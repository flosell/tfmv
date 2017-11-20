package main

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// https://github.com/palantir/tfjson/blob/57123411e29c8945cd8dc89b6237c8f6f31ddf6e/tfjson_test.go#L124-L132
func mustRun(t *testing.T, name string, arg ...string) {
	if _, err := exec.Command(name, arg...).Output(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			t.Fatal(string(exitError.Stderr))
		} else {
			t.Fatal(err)
		}
	}
}

func testModulePath(t *testing.T) string {
	module, err := filepath.Abs("test")
	assert.NoError(t, err)
	return module
}

func initModule(t *testing.T) {
	module := testModulePath(t)
	mustRun(t, "terraform", "init", module)
}

func generatePlan(t *testing.T) string {
	planFile, err := ioutil.TempFile("", "terraform-plan")
	assert.NoError(t, err)
	err = planFile.Close()
	assert.NoError(t, err)

	module := testModulePath(t)

	planPath := planFile.Name()
	mustRun(t, "terraform", "plan", "-out="+planPath, module)
	return planPath
}

func TestMissingPlan(t *testing.T) {
	_, err := getPlan("missing-file")
	assert.Error(t, err)
}

func TestSimplePlan(t *testing.T) {
	initModule(t)
	planPath := generatePlan(t)

	plan, err := getPlan(planPath)

	assert.NoError(t, err)
	assert.Len(t, plan.Diff.Modules, 1)
}

func TestChangesByType(t *testing.T) {
	initModule(t)
	planPath := generatePlan(t)
	plan, err := getPlan(planPath)
	assert.NoError(t, err)

	changesByType := getChangesByType(plan)

	assert.Equal(t, changesByType.GetTypes(), []ResourceType{"local_file"})
	changes := changesByType.Get("local_file")
	assert.Len(t, changes.Created, 1)
	assert.Len(t, changes.Destroyed, 0)
}
