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

func worker(pathCH <-chan string, wg *sync.WaitGroup, targetExt string, cryptor func(*lkf.Cryptor, []byte) int) {
	data := make([]byte, lkf.BlockSize*1024) // 512 Kb
	c := new(lkf.Cryptor)
	defer wg.Done()

pathGetting:
	for path := range pathCH {
		file, err := os.OpenFile(path, os.O_RDWR, 0644)
		if err != nil {
			log.Printf("Worker error: %s\n", err)
			continue pathGetting
		}

		for {
			n, err := file.Read(data)
			if n < lkf.BlockSize {
				if err == io.EOF {
					break
				}
				log.Printf("Worker error: %s\n", err)
				file.Close()
				continue pathGetting
			}
			cryptor(c, data[:n])

			// Moving on n bytes back, for record the decrypted/encrypted data
			if _, err := file.Seek(-int64(n), io.SeekCurrent); err != nil {
				log.Printf("Worker error: %s\n", err)
				file.Close()
				continue pathGetting
			}

			if _, err := file.Write(data[:n]); err != nil {
				log.Printf("Worker error: %s\n", err)
				file.Close()
				continue pathGetting
			}
		}

		file.Close()
		if err := os.Rename(path, path[:len(path)-4]+targetExt); err != nil {
			log.Printf("Worker error: %s\n", err)
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
		log.Fatal("Указано неподдерживаемое действие. Должно быть decode или encode")
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

	start := time.Now()
	if err := filepath.Walk(args[2], walker); err != nil {
		log.Printf("Filewalker: %s\n", err)
	}

	close(pathCH)
	wg.Wait()

	log.Printf("Обработано %d файлов за %v\n", fileCounter, time.Since(start))
}
