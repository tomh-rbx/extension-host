// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package shutdown

var (
	commandShutdownInstance Shutdown = newCommandShutdown()
	syscallShutdownInstance Shutdown = newSyscallShutdown()
)

type Shutdown interface {
	IsAvailable() bool
	Shutdown() error
	Reboot() error
	Name() string
}

func Get() Shutdown {
	if commandShutdownInstance.IsAvailable() {
		return commandShutdownInstance
	}
	if syscallShutdownInstance.IsAvailable() {
		return syscallShutdownInstance
	}
	return nil
}
