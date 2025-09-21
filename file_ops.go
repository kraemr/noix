package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func check(err error) {
	if err != nil {
		fmt.Printf("Error : %s", err.Error())
	}
}

func pathExists(path string) bool {
	fInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fInfo != nil
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

func buildRootPath(conf tCONFIG) string {
	return conf.Root + conf.Name
}

func createSymLink(old string, new string) {
	if debug == false {
		os.Symlink(old, new)
	} else {
		fmt.Printf("createSymLink %s %s\n", old, new)
	}
}

func IsFile(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		//fmt.Println(err)
		return false
	}
	return !fi.Mode().IsDir()
}

func createDirsIfMissing(filePath string) {
	if debug == true {
		fmt.Printf("createDirsIfMissing: %s  \n", filePath)
		return
	}

	parts := strings.Split(filePath, "/")
	directory := ""
	if len(parts) < 2 {
		return
	}
	for k := 0; k < len(parts)-1; k++ {
		directory += parts[k] + "/"
	}
	os.MkdirAll(directory, os.ModePerm)
}

func touch(path string, info os.FileInfo) (*os.File, error) {
	destFile, err := os.OpenFile(path,
		os.O_CREATE|os.O_RDWR|os.O_TRUNC,
		info.Mode())

	return destFile, err
}

func isSymbolicLink(path string) bool {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fileInfo.Mode()&os.ModeSymlink != 0
}

func copyFile(srcPath string, destPath string) {
	if debug {
		fmt.Printf("copyFile src:%s dst:%s\n", srcPath, destPath)
		return
	}

	srcFile, err := os.Open(srcPath)
	check(err)
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		check(err)
	}
	destFile, err := touch(destPath, info) // creates if file doesn't exist
	check(err)
	defer destFile.Close()
	if err := os.Chmod(destPath, info.Mode()); err != nil {
		check(err)
	}
	_, err = io.Copy(destFile, srcFile)
	check(err)
	err = destFile.Sync()
	check(err)
}

func recursePaths(srcPath string, root string, recursion_level int) {
	if recursion_level >= 4 {
		return
	}

	if isSymbolicLink(srcPath) {
		link, err := os.Readlink(srcPath)
		check(err)
		createDirsIfMissing(root + srcPath)
		err = os.Symlink(link, root+srcPath)
		check(err)
		realpath, err := filepath.EvalSymlinks(srcPath)
		check(err)
		if len(realpath) > 0 {
			recursePaths(realpath, root, recursion_level+1)
		}
		return
	}

	if !IsFile(srcPath) {
		entries, _ := os.ReadDir(srcPath)
		for i := 0; i < len(entries); i++ {
			if entries[i].IsDir() {
				os.MkdirAll(root+srcPath+"/"+entries[i].Name(), os.ModePerm)
			}
			recursePaths(srcPath+"/"+entries[i].Name(), root, recursion_level+1)
		}
		return
	}

	if IsFile(srcPath) {
		createDirsIfMissing(root + srcPath)
		copyFile(srcPath, root+srcPath)
	}

}

func SyncPaths(conf tCONFIG) {
	rootPath := buildRootPath(conf)
	for i := 0; i < len(conf.Sync_paths); i++ {
		srcPath := conf.Sync_paths[i]
		recursePaths(srcPath, rootPath, 0)
	}
}

func MakeSymLinks(conf tCONFIG) {
	chroot_path := buildRootPath(conf)

	for i := 0; i < len(conf.Sym_links); i++ {
		if debug != true {
			//fmt.Println("makeSymLink: ", conf.Sym_links[i][1], chroot_path+conf.Sym_links[i][0])
			createSymLink(conf.Sym_links[i][1], chroot_path+conf.Sym_links[i][0])
		} else {
			fmt.Printf("makeSymLinks os.Symlink: old %s new %s \n", conf.Sym_links[i][1], chroot_path+"/"+conf.Sym_links[i][0])
		}
	}

}
