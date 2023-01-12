// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package e2e is created because:
//
// During test setup, SetupLocalRegistry is called,
// and container tests need to be run before VM tests
// so that the local registry won't be removed by VM tests,
// and container tests can use the images stored in it.
//
// However, by default Ginkgo does not guarantee the order in which specs run [1],
// and Ginkgo doc says that "You should only ever call RunSpecs once" [2],
// which means that we need two binaries that run VM tests and container tests respectively.
//
// We could add ginkgo.Ordered to the top level Ginkgo container,
// but that will make every single spec to run in the order they are defined:
// "Any container nodes nested within a container node will automatically be considered Ordered
// and there is no way to mark a node within an Ordered container as "not Ordered"" [1],
// and we don't want that because it can hide implicit dependencies among tests,
// while each test should be independent unless specially designed.
//
// As a result, we split the tests into `e2e/vm` and `e2e/container` packages
// and extract the common logic into this package.
//
// [1] https://onsi.github.io/ginkgo/#ordered-containers.
// [2] https://onsi.github.io/ginkgo/#mental-model-go-modules-packages-and-tests
package e2e

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/runfinch/common-tests/option"
)

// InstalledTestSubject is the test subject when Finch is installed.
const InstalledTestSubject = "finch"

// Installed indicates whether the tests are run against installed application.
var Installed = flag.Bool("installed", false, "the flag to show whether the tests are run against installed application")

// CreateOption creates an option for running e2e tests.
func CreateOption() (*option.Option, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get the current working directory: %w", err)
	}

	subject := filepath.Join(wd, "../../_output/bin/finch")
	if *Installed {
		subject = InstalledTestSubject
	}

	o, err := option.New([]string{subject})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a testing option: %w", err)
	}
	return o, nil
}
