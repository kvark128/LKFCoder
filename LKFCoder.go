// Command-line utility for encoding/decoding LKF files.
package main

import (
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kvark128/lkf"
)

func FileCryptor(path string, cryptor func(*lkf.Cryptor, []byte) int) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, lkf.BlockSize*1024) // 512 Kb
	c := new(lkf.Cryptor)

	for err == nil {
		var n int
		n, err = io.ReadFull(f, buf)
		if n < lkf.BlockSize {
			break
		}

		// Encrypt or decrypt the read data
		cryptor(c, buf[:n])

		// Moving on n bytes back, for record the decrypted/encrypted data
		if _, err := f.Seek(-int64(n), io.SeekCurrent); err != nil {
			return err
		}

		if _, err := f.Write(buf[:n]); err != nil {
			return err
		}
	}

	if err == io.ErrUnexpectedEOF {
		// Just an end of the file. Skip this error
		err = nil
	}
	return err
}

func worker(pathCH <-chan string, wg *sync.WaitGroup, targetExt string, cryptor func(*lkf.Cryptor, []byte) int) {
	defer wg.Done()
	for path := range pathCH {
		targetPath := path[:len(path)-4] + targetExt
		tmpPath := targetPath + ".tmp"
		if err := os.Rename(path, tmpPath); err != nil {
			log.Printf("worker: %v", err)
			continue
		}

		if err := FileCryptor(tmpPath, cryptor); err != nil {
			log.Printf("worker: %v", err)
			continue
		}

		if err := os.Rename(tmpPath, targetPath); err != nil {
			log.Printf("worker: %v", err)
		}
	}
}

func main() {
	var args = []string{2: "./"}
	copy(args, os.Args)

	var fileCounter int
	var srcExt, targetExt string
	wg := new(sync.WaitGroup)
	pathCH := make(chan string)
	var cryptor func(*lkf.Cryptor, []byte) int

	log.SetFlags(0)
	switch args[1] {
	case "decode":
		cryptor = func(c *lkf.Cryptor, data []byte) int { return c.Decrypt(data, data) }
		srcExt = ".lkf"
		targetExt = ".mp3"
	case "encode":
		cryptor = func(c *lkf.Cryptor, data []byte) int { return c.Encrypt(data, data) }
		srcExt = ".mp3"
		targetExt = ".lkf"
	default:
		log.Fatal("An unsupported action is specified. Must be decode or encode")
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

	log.Printf("Please wait...\n")
	start := time.Now()
	if err := filepath.WalkDir(args[2], walker); err != nil {
		log.Printf("Filewalker: %s\n", err)
	}

	close(pathCH)
	wg.Wait()
	log.Printf("Processed %d *%s files in %v\n", fileCounter, srcExt, time.Since(start))
}
