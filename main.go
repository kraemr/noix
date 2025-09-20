package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"golang.org/x/sys/unix"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type (
	tCONFIG struct {
		Name        string
		Bootable    bool
		Immutable   bool
		Bind_mounts []string
		Sync_paths  []string
		Create_dirs []string
		Sym_links   [][2]string
	}
)

var debug bool = false
var max_recursion int = 10

func check(err error) {
	if err != nil {
		fmt.Println("Error : %s", err.Error())
		//   os.Exit(1)
	}
}

func pathExists(path string) bool {
	fInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fInfo != nil
}

func bindMounts(conf tCONFIG) {
	chrootPath := buildRootPath(conf.Name)
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

func buildRootPath(name string) string {
	return "/etc/noix/" + name
}

func createSymLink(old string, new string) {
	if debug == false {
		os.Symlink(old, new)
	} else {
		fmt.Printf("createSymLink %s %s\n", old, new)
	}
}

func FilePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func isFile(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Println(err)
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

func recreateSymlink(root string, path string, recursionLevel int) string {
	link, _ := os.Readlink(path)
	srcPath := path

	if recursionLevel > 8 {
		fmt.Printf("max recursion level reached %s \n", path)
		return ""
	}

	if len(link) > 0 { // Output of os.Readlink is OS-dependent...
		realpath, err := filepath.EvalSymlinks(srcPath)

		if err != nil {
			return ""
		}
		srcPath = realpath

		createSymLink(link, root+path)

		// if path doesnt exist in out "store"
		// and if its a dir create it
		if !pathExists(root+srcPath) && !isFile(srcPath) {
			os.MkdirAll(root+srcPath, os.ModePerm)
		}

		if !pathExists(root+srcPath) && isFile(srcPath) {
			//fmt.Println("!pathExists(root+srcPath) && isFile(srcPath): " + root + srcPath)
			createDirsIfMissing(root + srcPath)
			copyFile(srcPath, root+path)
		}

		if !isFile(srcPath) {
			paths, _ := FilePathWalkDir(srcPath)
			for i := 0; i < len(paths); i++ {
				recreateSymlink(root, paths[i], recursionLevel+1)
			}
		}

		return srcPath
	}
	return ""
}

func copy(src *os.File, dest *os.File) error {
	_, err := io.Copy(dest, src)
	return err
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
	written, err := io.Copy(destFile, srcFile)
	fmt.Println(written)
	//copy(srcFile, destFile) // check first var for number of bytes copied
	check(err)
	err = destFile.Sync()
	check(err)
}

func copyPath(name string, paths []string) {
	chroot_path := buildRootPath(name)
	for j := 0; j < len(paths); j++ {
		srcPath := paths[j]
		realpath, _ := filepath.EvalSymlinks(srcPath)
		srcPath = realpath

		for len(recreateSymlink(chroot_path, srcPath, 0)) > 0 { // Output of os.Readlink is OS-dependent...
			realpath, _ = filepath.EvalSymlinks(srcPath)
			srcPath = realpath
		}

		var fullPath string
		fullPath = chroot_path + srcPath
		if !isFile(paths[j]) {
			os.MkdirAll(fullPath, os.ModePerm)
		}

		parts := strings.Split(fullPath, "/")
		directory := ""
		if len(parts) < 2 {
			break
		}

		for k := 0; k < len(parts)-1; k++ {
			directory += parts[k] + "/"
		}
		os.MkdirAll(directory, os.ModePerm)
		if isFile(paths[j]) {
			copyFile(srcPath, fullPath)
		}
	}
}

func recursePaths(srcPath string, root string, recursion_level int) {
	if recursion_level >= 10 {
		return
	}

	if isSymbolicLink(srcPath) {
		link, err := os.Readlink(srcPath)
		check(err)
		fmt.Println(link)
		createDirsIfMissing(root + srcPath)
		err = os.Symlink(link, root+srcPath)
		check(err)
		realpath, err := filepath.EvalSymlinks(srcPath)
		check(err)
		if len(realpath) > 0 {
			recursePaths(realpath, root, recursion_level+1)
		}
	}

	if !isFile(srcPath) {
		entries, _ := os.ReadDir(srcPath)
		for i := 0; i < len(entries); i++ {
			if entries[i].IsDir() {
				os.MkdirAll(root+srcPath+"/"+entries[i].Name(), os.ModePerm)
			}
			recursePaths(srcPath+"/"+entries[i].Name(), root, recursion_level+1)
		}

	}

	if isFile(srcPath) {
		copyFile(srcPath, root+srcPath)
	}

}

func syncPaths(conf tCONFIG) {
	rootPath := buildRootPath(conf.Name)

	for i := 0; i < len(conf.Sync_paths); i++ {
		srcPath := conf.Sync_paths[i]
		recursePaths(srcPath, rootPath, 0)
	}

	/*
		chroot_path := "/etc/noix/" + conf.Name
		for i := 0; i < len(conf.Sync_paths); i++ {
			fmt.Println(conf.Sync_paths[i])
			link, err := os.Readlink(conf.Sync_paths[i])
			srcPath := conf.Sync_paths[i]
			if len(link) > 0 { // Output of os.Readlink is OS-dependent...
				realpath, _ := filepath.EvalSymlinks(srcPath)
				srcPath = realpath
				if err != nil {
					fmt.Println(err.Error())
				}
			}
			recreateSymlink(chroot_path, conf.Sync_paths[i], 0)
			paths, err := FilePathWalkDir(srcPath)
			if err != nil {
				check(err)
			}
			copyPath(conf.Name, paths)
		}*/
}

// switches to the root
func activate(conf tCONFIG) {

}

func makeSymLinks(conf tCONFIG) {
	chroot_path := buildRootPath(conf.Name)

	for i := 0; i < len(conf.Sym_links); i++ {
		if debug != true {
			createSymLink(conf.Sym_links[i][1], chroot_path+"/"+conf.Sym_links[i][0])
		} else {
			fmt.Printf("makeSymLinks os.Symlink: old %s new %s \n", conf.Sym_links[i][1], chroot_path+"/"+conf.Sym_links[i][0])
		}
	}

}

func createChroot(name string) {
	if !pathExists("/etc/noix") {
		err := os.MkdirAll("/etc/noix", os.ModePerm)
		if err != nil {
			fmt.Println("error in createChroot: mkdir -p /etc/noix missing permissions to create /etc/noix")
		}
	}
	path := fmt.Sprintf("/etc/noix/%s", name)
	err := os.Mkdir(path, os.ModePerm)
	if err != nil {
		fmt.Printf("error in createChroot: Failed to create directory %s\n", path)
	}
}

func main() {
	if len(os.Args) < 3 {
		copyExplanation := "copy copies all paths listed under sync_paths\n"
		copyUsage := "$ noix copy config.toml\n"
		bindExplanation := "bind creates mounts according to your config in bind_mounts\n"
		bindMounts := "$ noix bind config.toml\n"
		linkExplanation := "link creates symlinks as specified in sym_links\n"
		makeSymLinks := "$ noix link config.toml\n"
		fmt.Printf("%s%s%s%s%s%s",
			copyExplanation, copyUsage,
			bindExplanation, bindMounts,
			linkExplanation, makeSymLinks,
		)
		return
	}

	if len(os.Args) > 3 && os.Args[3] == "debug" {
		debug = true
	}
	var config tCONFIG
	_, _ = toml.DecodeFile(os.Args[2], &config)

	if os.Args[1] == "build" || os.Args[1] == "-b" {
		createChroot(config.Name)
		makeSymLinks(config)
		syncPaths(config)
		bindMounts(config)

	}
	if os.Args[1] == "copy" || os.Args[1] == "-c" {
		syncPaths(config)
	}

	if os.Args[1] == "bind" || os.Args[1] == "-bi" {
		bindMounts(config)
	}

	if os.Args[1] == "link" || os.Args[1] == "-li" {
		makeSymLinks(config)
	}
}
