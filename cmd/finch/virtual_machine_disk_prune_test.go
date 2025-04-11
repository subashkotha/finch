// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build darwin || windows

package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/runfinch/finch/pkg/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDiskPruneVMCommand(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		mockSvc func(*mocks.NerdctlCmdCreator, *mocks.Command, *mocks.Logger, *gomock.Controller)
		wantErr error
	}{
		{
			name: "successful disk prune",
			mockSvc: func(ncc *mocks.NerdctlCmdCreator, cmd *mocks.Command, logger *mocks.Logger, ctrl *gomock.Controller) {
				// Mock VM status check
				getVMStatusC := mocks.NewCommand(ctrl)
				ncc.EXPECT().CreateWithoutStdio("ls", "-f", "{{.Status}}", limaInstanceName).Return(getVMStatusC)
				getVMStatusC.EXPECT().Output().Return([]byte("Running"), nil)
				logger.EXPECT().Debugf("Status of virtual machine: %s", "Running")

				// Mock system prune
				pruneCmd := mocks.NewCommand(ctrl)
				ncc.EXPECT().CreateWithoutStdio("shell", limaInstanceName, "sudo", "-E", "nerdctl", "system", "prune", "-a", "-f").Return(pruneCmd)
				pruneCmd.EXPECT().CombinedOutput().Return([]byte(""), nil)
				logger.EXPECT().Info("Running system prune to remove unused containers, images, volumes, and networks...")
				logger.EXPECT().Info("System prune completed successfully")

				// Mock fstrim
				trimCmd := mocks.NewCommand(ctrl)
				ncc.EXPECT().CreateWithoutStdio("shell", limaInstanceName, "sudo", "fstrim", "-a").Return(trimCmd)
				trimCmd.EXPECT().CombinedOutput().Return([]byte(""), nil)
				logger.EXPECT().Info("Running fstrim to reclaim disk space...")
				logger.EXPECT().Info("Disk space reclaimed successfully")
			},
			wantErr: nil,
		},
		{
			name: "VM not running",
			mockSvc: func(ncc *mocks.NerdctlCmdCreator, cmd *mocks.Command, logger *mocks.Logger, ctrl *gomock.Controller) {
				getVMStatusC := mocks.NewCommand(ctrl)
				ncc.EXPECT().CreateWithoutStdio("ls", "-f", "{{.Status}}", limaInstanceName).Return(getVMStatusC)
				getVMStatusC.EXPECT().Output().Return([]byte("Stopped"), nil)
				logger.EXPECT().Debugf("Status of virtual machine: %s", "Stopped")
			},
			wantErr: fmt.Errorf("the instance %q is not running, run `finch %s start` to start the instance",
				limaInstanceName, virtualMachineRootCmd),
		},
		{
			name: "system prune fails",
			mockSvc: func(ncc *mocks.NerdctlCmdCreator, cmd *mocks.Command, logger *mocks.Logger, ctrl *gomock.Controller) {
				getVMStatusC := mocks.NewCommand(ctrl)
				ncc.EXPECT().CreateWithoutStdio("ls", "-f", "{{.Status}}", limaInstanceName).Return(getVMStatusC)
				getVMStatusC.EXPECT().Output().Return([]byte("Running"), nil)
				logger.EXPECT().Debugf("Status of virtual machine: %s", "Running")

				pruneCmd := mocks.NewCommand(ctrl)
				ncc.EXPECT().CreateWithoutStdio("shell", limaInstanceName, "sudo", "-E", "nerdctl", "system", "prune", "-a", "-f").Return(pruneCmd)
				pruneCmd.EXPECT().CombinedOutput().Return([]byte(""), errors.New("prune failed"))
				logger.EXPECT().Info("Running system prune to remove unused containers, images, volumes, and networks...")
				logger.EXPECT().Errorf("System prune failed: %v\n%s", errors.New("prune failed"), "")
			},
			wantErr: errors.New("prune failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			ncc := mocks.NewNerdctlCmdCreator(ctrl)
			cmd := mocks.NewCommand(ctrl)
			logger := mocks.NewLogger(ctrl)
			tc.mockSvc(ncc, cmd, logger, ctrl)

			err := newDiskPruneVMAction(ncc, logger).run()
			assert.Equal(t, err, tc.wantErr)
		})
	}
}
