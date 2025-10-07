package main

import (
	"golang.org/x/sys/unix"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func run(newRoot string, newRootExec string) {
	//syscall.* constants, not unix.*
	flags := syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID
	cmd := exec.Command("/proc/self/exe", append([]string{"child", newRoot, newRootExec}, os.Args[1:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: uintptr(flags), // ← cast added here
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
		GidMappingsEnableSetgroups: false,
	}

	if len(os.Args) > 1 && os.Args[1] == "child" {
		runChild(newRoot, newRootExec)
		return
	}

	if err := cmd.Run(); err != nil {
		log.Fatalf("unshare exec failed: %v", err)
	}
}

func runChild(newRoot string, newRootExec string) {
	absRoot, err := filepath.Abs(newRoot)
	if err != nil {
		log.Fatalf("cannot get absolute path: %v", err)
	}
	// Bind-mount newRoot to make it a mount point
	if err := unix.Mount(absRoot, absRoot, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		log.Fatalf("bind mount failed: %v", err)
	}

	// pivot_root into newRoot
	putOld := filepath.Join(absRoot, ".pivot_old")
	if err := os.MkdirAll(putOld, 0700); err != nil {
		log.Fatalf("mkdir .pivot_old failed: %v", err)
	}

	if err := unix.PivotRoot(absRoot, putOld); err != nil {
		log.Fatalf("pivot_root failed: %v", err)
	}
	if err := os.Chdir("/"); err != nil {
		log.Fatalf("chdir / failed: %v", err)
	}

	if err := unix.Unmount("/.pivot_old", unix.MNT_DETACH); err != nil {
		log.Printf("warning: unmount old root failed: %v", err)
	}

	os.RemoveAll("/.pivot_old")
	os.Clearenv()
	os.Setenv("PATH", "/bin:/usr/bin:/sbin")

	child := exec.Command(newRootExec)
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr

	if err := child.Run(); err != nil {
		log.Fatalf("failed to run %s: %v", newRootExec, err)
	}

}
