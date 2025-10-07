package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
	"os"
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
		Include     []string
		Is_loaded   bool
		Base        []string
		Use_Overlay bool
	}
)

var debug bool = false

func createChroot(config tCONFIG) {
	if !pathExists(config.Root + config.Name) {
		err := os.MkdirAll(config.Root+config.Name, os.ModePerm)
		if err != nil {
			fmt.Println("error in createChroot: mkdir -p /etc/noix missing permissions to create /etc/noix")
		}
	}
}

// TODO write metadata to know how a chroot was built, when etc
func writeBuildRecipe() {
	fileName := "noix_recipe"
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	// Make sure to close the file at the end
	defer file.Close()

	for i := 0; i < len(os.Args); i++ {
		_, err = file.WriteString(os.Args[i])
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}

	fmt.Println("File written successfully!")
}

func handleBuild(config tCONFIG) {
	createChroot(config)
	if len(config.Base) > 0 {
		for i := 0; i < len(config.Base); i++ {
			err := untarGz(config.Base[i], buildRootPath(config))
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	}
	MakeSymLinks(config)
	SyncPaths(config)
	BindMounts(config)
	if config.Use_Overlay {
		CreateOverlayFs(config, buildRootPath(config))
	}

}

func handleDecompress(config tCONFIG, out_archive_path *string) {

}

func handleCompress(config tCONFIG, out_archive_path *string) {
	if out_archive_path == nil {
		log.Fatal("archive path was nil")
	}
	out, err := os.Create(*out_archive_path)
	if err != nil {
		log.Fatalln("Error writing archive:", err)
	}
	defer out.Close()

	CreateArchive(config.Sync_paths, out)

	if err != nil {
		log.Fatalln("Error creating archive:", err)
	}

	fmt.Println("Archive created successfully")
}

func handleCreateDir(config tCONFIG) {
	CreateDirs(config)
}

func handleCopy(config tCONFIG) {
	SyncPaths(config)

}

func handleBind(config tCONFIG) {
	BindMounts(config)

}

func handleSymLink(config tCONFIG) {
	MakeSymLinks(config)
}

func handleActivate(config tCONFIG) {

}

func main() {
	if len(os.Args) > 3 && os.Args[1] == "child" {
		run(os.Args[2], os.Args[3])
	}

	cmd := flag.String("c", "help", "Command to execute, available(build, compress, create, copy, bind, link)")
	// Path to the config
	config_path := flag.String("config", "", "Your Configuration File")
	// path to the chroot or tar output
	out_path := flag.String("o", "out.tar.gz", "Path to the new archive or chroot")
	// override_name for the chroot (optional)
	override_name := flag.String("override-name", "", "You can override the name of the image")
	// when true an overlay_fs is created when a base image is used
	// where the base image's will be in lower (readonly), saving on space as it is not copied into the chroot
	// https://wiki.archlinux.org/title/Overlay_filesystem
	use_overlay_fs := flag.Bool("overlay", false, "")
	// Path to the created chroot to be chrooted
	activate_path := flag.String("chroot", "", "Path of chroot to activate")
	// When using this the uid and gid is dropped to 65534 (nobody) inside the chroot
	drop_root := flag.Bool("drop-privs", false, "Drop uid and gid (rootless)")
	custom_exec := flag.String("exec", "/bin/sh", "Path of program to exec in container")
	flag.Parse()
	var config tCONFIG
	if len(*config_path) != 0 {
		_, err := toml.DecodeFile(*config_path, &config)
		if err != nil {
			fmt.Println("Error Decoding Path:", config_path)
			os.Exit(1)
		}
		config.Is_loaded = true
	}
	config.Use_Overlay = *use_overlay_fs
	if len(*override_name) != 0 {
		config.Name = *override_name
	}

	switch *cmd {
	case "build":
		handleBuild(config)
	case "compress":
		handleCompress(config, out_path)
	case "activate_temp": // decompress file to /tmp and switch
	case "decompress":
		handleDecompress(config, out_path)
	case "create":
		handleCreateDir(config)
	case "copy":
		handleCopy(config)
	case "bind":
		handleBind(config)
	case "link":
		handleSymLink(config)
	case "activate":
		ActivateChroot(*activate_path, *drop_root)
	case "run":
		run(*activate_path, *custom_exec)
	}
}
