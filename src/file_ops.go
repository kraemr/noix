package main

import (
	"fmt"
	"io"
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

func buildRootPath(conf tCONFIG) string {
	if conf.Use_Overlay {
		return conf.Root + conf.Name + "/lower"
	} else {
		return conf.Root + conf.Name + "/chroot"
	}

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

func CreateDirs(config tCONFIG) {
	for i := 0; i < len(config.Create_dirs); i++ {
		dirName := config.Create_dirs[i]
		err := os.Mkdir(dirName, 0755)
		if err != nil {
			if os.IsExist(err) {
				fmt.Println("Directory already exists")
			} else {
				fmt.Println("Error creating directory:", err)
			}
			return
		}

		fmt.Println("Empty directory created:", dirName)
	}
}

func GetWorkingDir() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	return exPath
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

func BuildPath(dirPath string, name string) string {
	if len(strings.Split(dirPath, "/")) == 1 {
		return dirPath + name
	} else {
		return dirPath + "/" + name
	}
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

	info, err := os.Lstat(srcPath)
	if err != nil {
		return
	}
	mode := info.Mode()

	switch {
	case mode.IsRegular():
		fmt.Println("regular file")
		createDirsIfMissing(root + srcPath)
		copyFile(srcPath, root+srcPath)
	case mode.IsDir():
		fmt.Println("directory")
		entries, _ := os.ReadDir(srcPath)
		for i := range entries {
			if entries[i].IsDir() {
				os.MkdirAll(root+srcPath+"/"+entries[i].Name(), os.ModePerm)
			}
			recursePaths(srcPath+"/"+entries[i].Name(), root, recursion_level+1)
		}
	case mode&os.ModeSymlink != 0:
		fmt.Println("symlink")
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
	case mode&os.ModeNamedPipe != 0:
		fmt.Println("named pipe (FIFO)")
	case mode&os.ModeSocket != 0:
		fmt.Println("socket")
	case mode&os.ModeDevice != 0:
		if mode&os.ModeCharDevice != 0 {
			fmt.Println("character device")
		} else {
			fmt.Println("block device")
		}
	default:
		fmt.Println("unknown/other")
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
			createSymLink(conf.Sym_links[i][1], chroot_path+conf.Sym_links[i][0])
		} else {
			fmt.Printf("makeSymLinks os.Symlink: old %s new %s \n", conf.Sym_links[i][1], chroot_path+"/"+conf.Sym_links[i][0])
		}
	}

}
