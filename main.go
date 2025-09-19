package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type (
	tCONFIG struct {
		Name string
		Bootable bool
		Immutable bool
		Bind_mounts []string
		Sync_paths []string
		Create_dirs []string
		Sym_links [][2]string
	}
)

func check(err error) {
    if err != nil {
       // fmt.Println("Error : %s", err.Error())
     //   os.Exit(1)
    }
}

func pathExists(path string) bool {
	fInfo,err := os.Stat(path)
        if err != nil {
		return false;
        }
	return fInfo != nil
}

func bindMounts(conf tCONFIG){

}

func buildRootPath(name string) string{
	return "/etc/noix/" + name
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
	return !fi.Mode().IsDir();
}

func createDirsIfMissing(filePath string) {
	parts := strings.Split(filePath, "/");
	directory := "";
	if len(parts) < 2 {
		return;
	}
	for k :=0; k < len(parts)-1;k++{
		directory +=  parts[k] + "/";
	}
	os.MkdirAll(directory,os.ModePerm);
}


func recreateSymlink(root string,path string,recursionLevel int) string {
		link, _ := os.Readlink(path);
		srcPath := path;

		if recursionLevel > 8 {
			fmt.Println("max recursion level reached");
			return "";
		}

		if len(link) > 0 { // Output of os.Readlink is OS-dependent...
			realpath,err := filepath.EvalSymlinks(srcPath);
			if err != nil {
				return "";
			}
			srcPath = realpath;



			err = os.Symlink(link, root + path);
			fmt.Println(root + path);
			if err != nil {
				fmt.Println(err.Error());
			}
			
			// if path doesnt exist in out "store" 
			// and if its a dir create it
			if !pathExists(root+srcPath) && !isFile(srcPath) {
				os.MkdirAll(root+srcPath,os.ModePerm)
			}

			if !pathExists(root+srcPath) && isFile(srcPath) {				
				fmt.Println("!pathExists(root+srcPath) && isFile(srcPath): " + root+srcPath)
				createDirsIfMissing(root+srcPath)
				copyFile(srcPath,root+path);
			}

			if !isFile(srcPath) {
				paths, _ := FilePathWalkDir(srcPath);
				for i:=0; i< len(paths);i++ {
					recreateSymlink(root,paths[i],recursionLevel+1);
				}
			}
			
			return srcPath
		}
		return ""
}

func copyFile(srcPath string, destPath string){
	srcFile, err := os.Open(srcPath)
    check(err)
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		check(err);
	}
	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode()) // creates if file doesn't exist
    check(err)
	defer destFile.Close()				
	if err := os.Chmod(destPath, info.Mode()); err != nil {
		check(err);
	}
	_, err = io.Copy(destFile, srcFile) // check first var for number of bytes copied
    check(err)
    err = destFile.Sync()
    check(err)

}

func copyPath(name string,paths []string) {
	chroot_path := buildRootPath(name);	
	for j :=0; j < len(paths); j++ {
			srcPath := paths[j];
			realpath,_ := filepath.EvalSymlinks(srcPath);
			srcPath = realpath;
		
			for len(recreateSymlink(chroot_path ,srcPath,0)) > 0 { // Output of os.Readlink is OS-dependent...
				realpath,_ = filepath.EvalSymlinks(srcPath);
				srcPath = realpath;
			}

			var fullPath string;
			fullPath = chroot_path + srcPath						
			if !isFile(paths[j]) {
				os.MkdirAll(fullPath,os.ModePerm);
			}

			parts := strings.Split(fullPath, "/");
			directory := "";
			if len(parts) < 2 {
				break;
			}

			for k :=0; k < len(parts)-1;k++{
				directory +=  parts[k] + "/";
			}
			os.MkdirAll(directory,os.ModePerm);			
			
			if isFile(paths[j]) {
				copyFile(srcPath,fullPath)				
			}
			
	}
}

func copyPaths(conf tCONFIG){
	chroot_path := "/etc/noix/" + conf.Name;	
	fmt.Printf("copyPaths: %d", len(conf.Sync_paths))
	for i := 0; i < len(conf.Sync_paths); i++ {		
		fmt.Println(conf.Sync_paths[i]);
		link, err := os.Readlink(conf.Sync_paths[i]);
		srcPath := conf.Sync_paths[i];
		if len(link) > 0 { // Output of os.Readlink is OS-dependent...
			realpath,_ := filepath.EvalSymlinks(srcPath);
			srcPath = realpath;
			if err != nil {
				fmt.Println(err.Error());
			}
		}
		recreateSymlink(chroot_path,conf.Sync_paths[i], 0);
		paths, err := FilePathWalkDir(srcPath);
		fmt.Println(srcPath)
		if err != nil {
			check(err);
		}			
		//fmt.Println(paths)
		copyPath(conf.Name,paths);		    		
	}
				
	}
	



// switches to the root
func activate(conf tCONFIG){

}

func makeSymLinks(conf tCONFIG){
	chroot_path := buildRootPath(conf.Name);
	for i:=0; i < len(conf.Sym_links); i++ {
		os.Symlink(conf.Sym_links[i][1], chroot_path + "/" + conf.Sym_links[i][0]);
	}

}

func hashFile(path string) ([]byte,error) {
  file, err := os.Open(path)
  if err != nil {
    log.Fatal(err)
    return nil,err
  }
  defer file.Close()

  hash := sha256.New()
  if _, err := io.Copy(hash, file); err != nil {
    log.Fatal(err)
    return nil, err
  }

  return hash.Sum(nil),nil
  
}

func createChroot(name string) {
	if !pathExists("/etc/noix") {
		err := os.MkdirAll("/etc/noix", os.ModePerm)
		if err != nil {
			fmt.Println("error in createChroot: mkdir -p /etc/noix missing permissions to create /etc/noix")
		}
	}
    path := fmt.Sprintf("/etc/noix/%s", name)
	err := os.Mkdir(path,os.ModePerm)
	if err != nil {
		fmt.Printf("error in createChroot: Failed to create directory %s\n", path);
	}
}



func main(){
   if len(os.Args) < 3 {
	return
   }
   
   if os.Args[1] == "build" || os.Args[1] == "-b" {
	var config tCONFIG
	_,_ = toml.DecodeFile(os.Args[2],&config)
   	createChroot(config.Name)
	makeSymLinks(config);
	copyPaths(config)
	//chroot_path := "/etc/noix/" + config.Name;	
	
	// path := "/lib64"
	// for len(path) > 0 {
	// 	path = recreateSymlink(chroot_path,path)
	// }
	// fmt.Println(path)
	//copyPaths(config)
	// realpath,_ := filepath.EvalSymlinks("/lib64");
	// fmt.Println(realpath);

	// paths, err := FilePathWalkDir(realpath);
	// check(err)
	// fmt.Println(paths);
	

	// realpath,_ = filepath.EvalSymlinks(paths[0]);
	// fmt.Println(realpath);


   }
}
