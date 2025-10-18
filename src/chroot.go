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
		Cloneflags: uintptr(flags),
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1}, // allow 0-65535 inside
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

	// Make mounts private so they don't leak or get blocked by shared propagation
	if err := unix.Mount("", "/", "", unix.MS_PRIVATE|unix.MS_REC, ""); err != nil {
		log.Fatalf("failed to make mounts private: %v", err)
	}

	// Bind-mount newRoot to make it a mount point
	if err := unix.Mount(absRoot, absRoot, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		log.Fatalf("bind mount failed: %v", err)
	}

	// Prepare /proc and /dev in the new root before pivot
	os.MkdirAll(filepath.Join(absRoot, "proc"), 0555)
	os.MkdirAll(filepath.Join(absRoot, "dev"), 0755)

	// Mount /proc
	if err := unix.Mount("proc", filepath.Join(absRoot, "proc"), "proc", 0, ""); err != nil {
		log.Printf("mount /proc failed: %v", err)
	}

	// Mount tmpfs for /dev
	if err := unix.Mount("tmpfs", filepath.Join(absRoot, "dev"), "tmpfs", 0, "mode=755"); err != nil {
		log.Printf("mount /dev failed: %v", err)
	}

	// Create the device files
	devs := []string{"null", "zero", "tty", "random", "urandom"}
	for _, d := range devs {
		path := filepath.Join(absRoot, "dev", d)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			f, err := os.Create(path)
			if err == nil {
				f.Close()
			}
		}
	}

	// Bind-mount host device nodes into the new /dev
	binds := []string{"null", "zero", "tty", "random", "urandom"}
	for _, d := range binds {
		src := filepath.Join("/dev", d)
		dst := filepath.Join(absRoot, "dev", d)
		if err := unix.Mount(src, dst, "", unix.MS_BIND, ""); err != nil {
			log.Printf("bind %s failed: %v", d, err)
		}
	}

	// Optional: /dev/pts for TTY support
	os.MkdirAll(filepath.Join(absRoot, "dev/pts"), 0755)
	if err := unix.Mount("devpts", filepath.Join(absRoot, "dev/pts"), "devpts", 0,
		"newinstance,ptmxmode=0666,mode=620,gid=5"); err != nil {
		log.Printf("mount /dev/pts failed (harmless on some kernels): %v", err)
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
	os.Setenv("HOME", "/root")

	child := exec.Command(newRootExec)
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr

	if err := child.Run(); err != nil {
		log.Fatalf("failed to run %s: %v", newRootExec, err)
	}
}
