package main

// A little Cursed, but this creates a wrapper for calling noix to be called from grub as the init program

import "os"
import "fmt"

func createWrapper(programPath string, wrapperPath string, chrootPath string) {
	wrapper := fmt.Sprintf("#!/bin/sh\n%s -c=pivot -pivot=%s", programPath, chrootPath)
	d1 := []byte(wrapper)
	err := os.WriteFile(wrapperPath, d1, 0755)
	check(err)
}
