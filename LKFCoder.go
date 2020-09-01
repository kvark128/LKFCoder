// Utility for encoding/decoding lkf files.
package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kvark128/lkf"
)

func coder(files <-chan *os.File, wg *sync.WaitGroup, targetExt string, cryptor func(*lkf.Cryptor, []byte) int, errors chan<- error) {
	data := make([]byte, lkf.BlockSizeInBytes*1024)
	c := new(lkf.Cryptor)
	defer wg.Done()
	for srcFile := range files {
		for {
			n, _ := srcFile.Read(data)
			if n < lkf.BlockSizeInBytes {
				break
			}
			cryptor(c, data[:n])

			// Moving on n bytes back, for record the decrypted/encrypted data
			if _, err := srcFile.Seek(-int64(n), 1); err != nil {
				errors <- err
				break
			}

			if _, err := srcFile.Write(data[:n]); err != nil {
				errors <- err
				break
			}
		}

		path := srcFile.Name()
		srcFile.Close()
		os.Rename(path, path[:len(path)-3]+targetExt)
	}
}

func main() {
	var args = []string{2: "./"}
	copy(args, os.Args)

	var counterFiles int
	var srcExt string
	wg := new(sync.WaitGroup)
	files := make(chan *os.File)
	errors := make(chan error)

	workerCreator := func(targetExt string, cryptor func(*lkf.Cryptor, []byte) int) {
		for n := runtime.NumCPU(); n > 0; n-- {
			wg.Add(1)
			go coder(files, wg, targetExt, cryptor, errors)
		}
	}

	log.SetFlags(0)
	switch args[1] {
	case "decode":
		workerCreator("mp3", func(c *lkf.Cryptor, data []byte) int { return c.Decrypt(data) })
		srcExt = ".lkf"
	case "encode":
		workerCreator("lkf", func(c *lkf.Cryptor, data []byte) int { return c.Encrypt(data) })
		srcExt = ".mp3"
	default:
		log.Fatal("Указано неподдерживаемое действие. Должно быть decode или encode")
	}

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.ToLower(filepath.Ext(path)) != srcExt {
			return err
		}

		srcFile, err := os.OpenFile(path, os.O_RDWR, 0644)
		if err != nil {
			return err
		}

		counterFiles++
		files <- srcFile
		return nil
	}

	go func() {
		if err := filepath.Walk(args[2], walker); err != nil {
			errors <- err
		}
		close(files)
		wg.Wait()
		close(errors)
	}()

	log.Println("Пожалуйста, подождите...")
	start := time.Now()
	for err := range errors {
		log.Println(err)
	}

	log.Printf("Обработано %d файлов за %v\n", counterFiles, time.Since(start))
}
