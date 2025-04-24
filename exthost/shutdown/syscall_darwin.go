// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

//go:build darwin

package shutdown

import (
	"github.com/rs/zerolog/log"
	"syscall"
	"time"
)

type syscallShutdown struct {
	sc func(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)
}

func newSyscallShutdown() Shutdown {
	return &syscallShutdown{sc: syscall.Syscall}
}

func (s *syscallShutdown) IsAvailable() bool {
	return true
}

func (s *syscallShutdown) Reboot() error {
	return s.runSyscall(syscall.SYS_REBOOT, "SYS_REBOOT")
}

func (s *syscallShutdown) Shutdown() error {
	return s.runSyscall(syscall.SYS_SHUTDOWN, "SYS_SHUTDOWN")
}

func (s *syscallShutdown) Name() string {
	return "syscall"
}

func (s *syscallShutdown) runSyscall(trap uintptr, name string) error {
	go func() {
		time.Sleep(3 * time.Second)
		if _, _, err := s.sc(trap, 0, 0, 0); err != 0 {
			log.Err(err).Msgf("failed %s", name)
		}
	}()
	return nil
}
