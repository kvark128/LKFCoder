// Command-line utility for encoding/decoding LKF files.
package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kvark128/lkf"
)

func FileCryptor(path string, cryptor func(*lkf.Cryptor, []byte) int) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}

	buf := make([]byte, lkf.BlockSize*1024) // 512 Kb
	c := new(lkf.Cryptor)
	var off int64

	for err == nil {
		err = func() error {
			n, err := io.ReadFull(f, buf)
			if err != nil && err != io.ErrUnexpectedEOF {
				// Fatal error when reading from file
				// Processing must be aborted immediately
				return err
			}

			// Encrypt or decrypt the read data
			// Processed data should be written back to the file instead of the previously read ones
			if np := cryptor(c, buf[:n]); np != 0 {
				if _, err := f.WriteAt(buf[:np], off); err != nil {
					// Fatal error when writing to file
					// Processing must be aborted immediately
					return err
				}
				off += int64(np)
			}
			return err
		}()
	}

	if err != io.ErrUnexpectedEOF {
		// Fatal error occurred while processing the file. Close the file and return this error
		f.Close()
		return err
	}

	// Just the end of the file with no errors. Closing it
	return f.Close()
}

func worker(pathCH <-chan string, wg *sync.WaitGroup, targetExt string, cryptor func(*lkf.Cryptor, []byte) int) {
	defer wg.Done()
	for path := range pathCH {
		targetPath := path[:len(path)-4] + targetExt
		tmpPath := targetPath + ".tmp"
		if err := os.Rename(path, tmpPath); err != nil {
			fmt.Printf("worker: %v\n", err)
			continue
		}

		if err := FileCryptor(tmpPath, cryptor); err != nil {
			fmt.Printf("worker: %v\n", err)
			continue
		}

		if err := os.Rename(tmpPath, targetPath); err != nil {
			fmt.Printf("worker: %v\n", err)
		}
	}
}

func main() {
	var action string
	var fileCounter int
	var srcExt, targetExt string
	var cryptor func(*lkf.Cryptor, []byte) int
	wg := new(sync.WaitGroup)
	pathCH := make(chan string)
	workingDir := "./"

	if len(os.Args) >= 2 {
		action = os.Args[1]
	}

	if len(os.Args) >= 3 {
		workingDir = os.Args[2]
	}

	switch action {
	case "decode":
		cryptor = func(c *lkf.Cryptor, data []byte) int { return c.Decrypt(data, data) }
		srcExt = ".lkf"
		targetExt = ".mp3"
	case "encode":
		cryptor = func(c *lkf.Cryptor, data []byte) int { return c.Encrypt(data, data) }
		srcExt = ".mp3"
		targetExt = ".lkf"
	default:
		fmt.Printf("Unsupported action specified\n")
		os.Exit(1)
	}

	for n := runtime.NumCPU(); n > 0; n-- {
		wg.Add(1)
		go worker(pathCH, wg, targetExt, cryptor)
	}

	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.ToLower(filepath.Ext(path)) != srcExt {
			return err
		}
		fileCounter++
		pathCH <- path
		return nil
	}

	start := time.Now()
	if err := filepath.WalkDir(workingDir, walker); err != nil {
		fmt.Printf("Filewalker: %v\n", err)
	}

	close(pathCH)
	wg.Wait()
	finish := time.Since(start)
	fmt.Printf("Processed %d *%s files in %v\n", fileCounter, srcExt, finish)
}
