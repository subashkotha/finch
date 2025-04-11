// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build darwin || windows

package main

import (
	"fmt"

	"github.com/runfinch/finch/pkg/command"
	"github.com/runfinch/finch/pkg/flog"
	"github.com/runfinch/finch/pkg/lima"

	"github.com/spf13/cobra"
)

func newDiskPruneVMCommand(limaCmdCreator command.NerdctlCmdCreator, logger flog.Logger) *cobra.Command {
	diskPruneVMCommand := &cobra.Command{
		Use:   "disk-prune",
		Short: "Prune unused disk space in the virtual machine",
		RunE:  newDiskPruneVMAction(limaCmdCreator, logger).runAdapter,
	}

	return diskPruneVMCommand
}

type diskPruneVMAction struct {
	creator command.NerdctlCmdCreator
	logger  flog.Logger
}

func newDiskPruneVMAction(creator command.NerdctlCmdCreator, logger flog.Logger) *diskPruneVMAction {
	return &diskPruneVMAction{creator: creator, logger: logger}
}

func (dpa *diskPruneVMAction) runAdapter(_ *cobra.Command, _ []string) error {
	return dpa.run()
}

func (dpa *diskPruneVMAction) run() error {
	// 1. Check if VM is running
	status, err := lima.GetVMStatus(dpa.creator, dpa.logger, limaInstanceName)
	if err != nil {
		return err
	}
	if status != lima.Running {
		return fmt.Errorf("the instance %q is not running, run `finch %s start` to start the instance",
			limaInstanceName, virtualMachineRootCmd)
	}

	// 2. Run system prune
	dpa.logger.Info("Running system prune to remove unused containers, images, volumes, and networks...")
	pruneCmd := dpa.creator.CreateWithoutStdio("shell", limaInstanceName, "sudo", "-E", "nerdctl", "system", "prune", "-a", "-f")
	logs, err := pruneCmd.CombinedOutput()
	if err != nil {
		dpa.logger.Errorf("System prune failed: %v\n%s", err, logs)
		return err
	}
	dpa.logger.Info("System prune completed successfully")

	// 3. Ask for confirmation before running fstrim
	fmt.Println("\nDisk Cleanup: Reclaim unused space from the VM's disk using fstrim.")
	fmt.Println("⚠️ Warning: Running fstrim frequently, or using the discard mount option,")
	fmt.Println("might negatively affect the lifespan of some SSDs.")
	fmt.Print("Proceed with disk trim? (y/N): ")

	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		dpa.logger.Info("Skipped disk trim. System prune completed.")
		return nil
	}

	// 4. Run fstrim
	dpa.logger.Info("Running fstrim to reclaim disk space...")
	trimCmd := dpa.creator.CreateWithoutStdio("shell", limaInstanceName, "sudo", "fstrim", "-a")
	logs, err = trimCmd.CombinedOutput()
	if err != nil {
		dpa.logger.Errorf("Fstrim failed: %v\n%s", err, logs)
		return err
	}
	dpa.logger.Info("Disk space reclaimed successfully")

	return nil
}
