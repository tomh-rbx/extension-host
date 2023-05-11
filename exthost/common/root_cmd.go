package common

import (
	"os/exec"
	"syscall"
)

func RunAsRoot(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: 0,
			Gid: 0,
		},
	}
	return cmd.Run()
}

func RunAsRootWithResult(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: 0,
			Gid: 0,
		},
	}
	return cmd.CombinedOutput()
}
