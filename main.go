package main

import (
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

	if len(os.Args) == 4 && os.Args[1] == "compress" {
		out, err := os.Create(os.Args[3])
		if err != nil {
			log.Fatalln("Error writing archive:", err)
		}
		defer out.Close()
		var f []string
		f = append(f, os.Args[2])
		CreateArchive(f, out)
		os.Exit(0)
	}

	var config tCONFIG
	_, _ = toml.DecodeFile(os.Args[2], &config)
	// Directly builds
	if os.Args[1] == "build" || os.Args[1] == "-b" {
		createChroot(config)
		MakeSymLinks(config)
		SyncPaths(config)
		BindMounts(config)
	}

	// recursively read directories and compress them
	if (os.Args[1] == "compress" || os.Args[1] == "-co") && os.Args[3] == "-s" {
		out, err := os.Create(os.Args[4])
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

	if os.Args[1] == "copy" || os.Args[1] == "-cp" {
		SyncPaths(config)
	}

	if os.Args[1] == "bind" || os.Args[1] == "-bi" {
		BindMounts(config)
	}

	if os.Args[1] == "link" || os.Args[1] == "-li" {
		MakeSymLinks(config)
	}
}
