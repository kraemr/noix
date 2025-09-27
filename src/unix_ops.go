package main

import "log"
import "os"
import "golang.org/x/sys/unix"
import "fmt"

// Always drops into /bin/sh TODO add option to exec arbitrary commands inside chroot and return instantly
func ActivateChroot(path string, drop_root bool) {
	newRoot := path + "/chroot"

	// Ensure target exists
	if _, err := os.Stat(newRoot); os.IsNotExist(err) {
		log.Fatalf("chroot target %s does not exist", newRoot)
	}

	// Perform chroot
	if err := unix.Chroot(newRoot); err != nil {
		log.Fatalf("chroot failed: %v", err)
	}

	// Change working directory to "/"
	if err := os.Chdir("/"); err != nil {
		log.Fatalf("failed to chdir to / after chroot: %v", err)
	}

	log.Println("Chroot successful, new root is:", newRoot)

	if drop_root {
		DropPrivs()
	}
	if err := unix.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ()); err != nil {
		log.Fatalf("exec failed: %v", err)
	}
}

func DropPrivs() {
	// Drop to nobody user (UID/GID = 65534 usually)
	if err := unix.Setgid(65534); err != nil {
		log.Fatalf("failed to drop GID: %v", err)
	}

	if err := unix.Setuid(65534); err != nil {
		log.Fatalf("failed to drop UID: %v", err)
	}

	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		log.Fatalf("failed to set no_new_privs: %v", err)
	}
}

func CreateOverlayFs(config tCONFIG, base string) error {
	lowerDir := base
	path := buildRootPath(config)
	path = removeTrailing(path, "/lower")
	upperDir := path + "/upper"
	if err := os.MkdirAll(upperDir, 0755); err != nil {
		log.Fatalf("failed to create target dir %s: %v", upperDir, err)
	}

	workDir := path + "/work"
	if err := os.MkdirAll(workDir, 0755); err != nil {
		log.Fatalf("failed to create target dir %s: %v", upperDir, err)
	}

	mergedDir := path + "/chroot"
	if err := os.MkdirAll(mergedDir, 0755); err != nil {
		log.Fatalf("failed to create target dir %s: %v", mergedDir, err)
	}
	options := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, upperDir, workDir)

	fmt.Println(options)

	if err := unix.Mount("overlay", mergedDir, "overlay", 0, options); err != nil {
		log.Fatalf("failed to mount overlayFS: %v", err)
		return err
	}

	return nil
}

func BindMounts(conf tCONFIG) {
	chrootPath := buildRootPath(conf)
	for i := 0; i < len(conf.Bind_mounts); i++ {
		dst := chrootPath + conf.Bind_mounts[i]

		if err := os.MkdirAll(dst, 0755); err != nil {
			log.Fatalf("failed to create target dir %s: %v", dst, err)
		}

		if err := unix.Mount(conf.Bind_mounts[i], dst, "", unix.MS_BIND, ""); err != nil {
			log.Fatalf("failed to bind mount %s -> %s: %v", conf.Bind_mounts[i], dst, err)
		}

		if err := unix.Mount("", dst, "", unix.MS_BIND|unix.MS_REMOUNT|unix.MS_RDONLY, ""); err != nil {
			log.Fatalf("failed to remount readonly %s: %v", dst, err)
		}
	}
}
