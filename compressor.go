package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
)

func CreateArchive(files []string, buf io.Writer) {
	gw := gzip.NewWriter(buf)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	rescursiveCreateArchive(files, tw, 0)

}

// shamelessly copied from https://www.arthurkoziel.com/writing-tar-gz-files-in-go/
func rescursiveCreateArchive(files []string, tw *tar.Writer, recursion_level int) error {
	if recursion_level > 4 {
		return errors.New("rescursiveCreateArchive: Max Recursion Level Reached")
	}
	// Iterate over files and add them to the tar archive
	for _, file := range files {

		if isSymbolicLink(file) {
			addSymlinkToArchive(tw, file)
			continue
		}

		if !IsFile(file) {
			entries, err := os.ReadDir(file)
			if err != nil {
				fmt.Println(err.Error())
			}
			var recurseFiles []string
			for i := 0; i < len(entries); i++ {
				recurseFiles = append(recurseFiles, file+"/"+entries[i].Name())
			}
			rescursiveCreateArchive(recurseFiles, tw, recursion_level+1)
		} else {
			err := addToArchive(tw, file)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func addSymlinkToArchive(tw *tar.Writer, filename string) error {
	// Use Lstat so we get info about the symlink itself, not its target
	info, err := os.Lstat(filename)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return &os.PathError{
			Op:   "addSymlinkToArchive",
			Path: filename,
			Err:  os.ErrInvalid,
		}
	}

	// Read the target of the symlink
	linkTarget, err := os.Readlink(filename)
	if err != nil {
		return err
	}

	// Create a tar header for the symlink
	header, err := tar.FileInfoHeader(info, linkTarget)
	if err != nil {
		return err
	}

	// Use the filename in the archive (optional: can be relative)
	header.Name = filename

	// Write the header (symlinks have no file contents)
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	return nil
}

func addToArchive(tw *tar.Writer, filename string) error {
	// Open the file which will be written into the archive
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get FileInfo about our file providing file size, mode, etc.
	info, err := file.Stat()
	if err != nil {
		return err
	}

	var link string = ""
	if info.Mode()&os.ModeSymlink != 0 {
		link, err = os.Readlink(filename)
		if err != nil {
			return err
		}

		// Create a tar Header from the FileInfo data
		header, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}

		// Use full path as name (FileInfoHeader only takes the basename)
		// If we don't do this the directory strucuture would
		// not be preserved
		// https://golang.org/src/archive/tar/common.go?#L626
		header.Name = filename
		// Write file header to the tar archive
		err = tw.WriteHeader(header)
		if err != nil {
			return err
		}

		// Copy file content to tar archive
		_, err = io.Copy(tw, file)
		if err != nil {
			return err
		}

		return nil
	}

	// Create a tar Header from the FileInfo data
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	// Use full path as name (FileInfoHeader only takes the basename)
	// If we don't do this the directory strucuture would
	// not be preserved
	// https://golang.org/src/archive/tar/common.go?#L626
	header.Name = filename
	// Write file header to the tar archive
	err = tw.WriteHeader(header)
	if err != nil {
		return err
	}

	// Copy file content to tar archive
	_, err = io.Copy(tw, file)
	if err != nil {
		return err
	}

	return nil

}
