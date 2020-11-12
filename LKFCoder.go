// Command-line utility for encoding/decoding LKF files.
package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kvark128/lkf"
)

func FillBuffer(r io.Reader, buf []byte) (int, error) {
	var rn int
	for {
		n, err := r.Read(buf[rn:])
		rn += n
		if err != nil {
			return rn, err
		}
		if rn == len(buf) {
			return rn, nil
		}
	}
}

func FileCryptor(f *os.File, cryptor func(*lkf.Cryptor, []byte) int) error {
	buf := make([]byte, lkf.BlockSize*1024) // 512 Kb
	c := new(lkf.Cryptor)

	for {
		n, err := FillBuffer(f, buf)
		if n < lkf.BlockSize {
			if err == io.EOF {
				return nil
			}
			return err
		}
		cryptor(c, buf[:n])

		// Moving on n bytes back, for record the decrypted/encrypted data
		if _, err := f.Seek(-int64(n), io.SeekCurrent); err != nil {
			return err
		}

		if _, err := f.Write(buf[:n]); err != nil {
			return err
		}
	}
}

func worker(pathCH <-chan string, wg *sync.WaitGroup, targetExt string, cryptor func(*lkf.Cryptor, []byte) int) {
	defer wg.Done()
	for path := range pathCH {
		f, err := os.OpenFile(path, os.O_RDWR, 0644)
		if err != nil {
			log.Printf("worker: %v", err)
			continue
		}

		err = FileCryptor(f, cryptor)
		f.Close()
		if err != nil {
			log.Printf("worker: %v", err)
			continue
		}

		if err := os.Rename(path, path[:len(path)-4]+targetExt); err != nil {
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

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.ToLower(filepath.Ext(path)) != srcExt {
			return err
		}
		fileCounter++
		pathCH <- path
		return nil
	}

	log.Printf("Please wait...\n")
	start := time.Now()
	if err := filepath.Walk(args[2], walker); err != nil {
		log.Printf("Filewalker: %s\n", err)
	}

	close(pathCH)
	wg.Wait()
	log.Printf("Processed %d *%s files in %v\n", fileCounter, srcExt, time.Since(start))
}
