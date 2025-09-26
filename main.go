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

func handleBuild(config tCONFIG) {
	createChroot(config)
	if len(config.Base) > 0 {
		for i := 0; i < len(config.Base); i++ {

			str := GetWorkingDir()
			fmt.Println(str)

			err := untarGz(config.Base[i], buildRootPath(config))
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	}
	MakeSymLinks(config)
	SyncPaths(config)
	BindMounts(config)
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
	cmd := flag.String("c", "help", "Command to execute, available(build, compress, create, copy, bind, link)")
	config_path := flag.String("config", "", "Your Configuration File")
	out_path := flag.String("o", "out.tar.gz", "Path to the new archive or chroot")
	flag.Parse()

	var config tCONFIG

	if len(*config_path) != 0 {
		_, err := toml.DecodeFile(*config_path, &config)
		if err != nil {
			fmt.Println("Error Decoding Path:", config_path)
			os.Exit(1)
		}
	}
	config.Is_loaded = true

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
	}
}
