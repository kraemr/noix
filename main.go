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
		Root        string
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

func bindMounts(conf tCONFIG) {
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

func recursePaths(srcPath string, root string, recursion_level int) {
	fmt.Printf("recursePaths: %s %d\n", srcPath, recursion_level)
	if recursion_level >= 4 {
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
		createDirsIfMissing(root + srcPath)
		copyFile(srcPath, root+srcPath)
	}

}

func syncPaths(conf tCONFIG) {
	rootPath := buildRootPath(conf)
	for i := 0; i < len(conf.Sync_paths); i++ {
		srcPath := conf.Sync_paths[i]
		recursePaths(srcPath, rootPath, 0)
	}
}

// switches to the root
func activate(conf tCONFIG) {

}

func makeSymLinks(conf tCONFIG) {
	chroot_path := buildRootPath(conf)

	for i := 0; i < len(conf.Sym_links); i++ {
		if debug != true {
			fmt.Println("makeSymLink: ", conf.Sym_links[i][1], chroot_path+conf.Sym_links[i][0])
			createSymLink(conf.Sym_links[i][1], chroot_path+conf.Sym_links[i][0])
		} else {
			fmt.Printf("makeSymLinks os.Symlink: old %s new %s \n", conf.Sym_links[i][1], chroot_path+"/"+conf.Sym_links[i][0])
		}
	}

}

func createChroot(config tCONFIG) {
	if !pathExists(config.Root + config.Name) {
		err := os.MkdirAll(config.Root+config.Name, os.ModePerm)
		if err != nil {
			fmt.Println("error in createChroot: mkdir -p /etc/noix missing permissions to create /etc/noix")
		}
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
		createChroot(config)
		makeSymLinks(config)
		syncPaths(config)
		bindMounts(config)

	}

	if os.Args[1] == "compress" || os.Args[1] == "-co" {
		files := []string{"/home/rob/minimal_bash"}
		out, err := os.Create("output.tar.gz")
		if err != nil {
			log.Fatalln("Error writing archive:", err)
		}
		defer out.Close()
		err = createArchive(files, out)
		if err != nil {
			log.Fatalln("Error creating archive:", err)
		}

		fmt.Println("Archive created successfully")
	}

	if os.Args[1] == "copy" || os.Args[1] == "-cp" {
		syncPaths(config)
	}

	if os.Args[1] == "bind" || os.Args[1] == "-bi" {
		bindMounts(config)
	}

	if os.Args[1] == "link" || os.Args[1] == "-li" {
		makeSymLinks(config)
	}
}
