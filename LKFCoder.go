// Utility for encoding/decoding lkf files.
// Used 3-pass block cipher XXTEA with 128-bit key and block size of 128 words.
package main

import (
	"encoding/binary"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	blockSize = 128 // The block size in words. Every word of 32 bit
	delta     = 0x9e3779b9
	version   = "0.4" // Program version
)

// The 128-bit key for encrypting/decrypting lkf files. It is divided into 4 parts of 32 bit each.
var key = [4]uint32{
	0x8ac14c27,
	0x42845ac1,
	0x136506bb,
	0x05d47c66,
}

func calcKey(leftWord, rightWord, r, k uint32) uint32 {
	n1 := (leftWord>>5 ^ rightWord<<2) + (rightWord>>3 ^ leftWord<<4)
	n2 := (key[(k&3)^(r>>2&3)] ^ leftWord) + (r ^ rightWord)
	return n1 ^ n2
}

// decoder function decrypts the lkf-file and writes the result to the source file, then changes the extension to .mp3
// Decoding occurs by blocks from the beginning of the file. If the end of file is less than block size, it remains as it is.
func decoder(files <-chan *os.File, wg *sync.WaitGroup, errors chan<- error) {
	var block = make([]uint32, blockSize)
	defer wg.Done()
	for srcFile := range files {
		for binary.Read(srcFile, binary.LittleEndian, block) == nil {
			for r := uint32(3); r != 0; r-- {
				for k := blockSize - 1; k >= 0; k-- {
					block[k] -= calcKey(block[(k-1)&(blockSize-1)], block[(k+1)&(blockSize-1)], r*delta, uint32(k))
				}
			}

			// Moving on 1 block back, for record the decoded block
			if _, err := srcFile.Seek(blockSize*-4, 1); err != nil {
				errors <- err
				break
			}

			if err := binary.Write(srcFile, binary.LittleEndian, block); err != nil {
				errors <- err
				break
			}
		}

		path := srcFile.Name()
		srcFile.Close()
		os.Rename(path, path[:len(path)-3]+"mp3")
	}
}

// encoder function encrypts the mp3-file and writes the result to the source file, then changes the extension to .lkf
// Encoding occurs by blocks from the beginning of the file. If the end of file is less than block size, it remains as it is.
func encoder(files <-chan *os.File, wg *sync.WaitGroup, errors chan<- error) {
	var block = make([]uint32, blockSize)
	defer wg.Done()
	for srcFile := range files {
		for binary.Read(srcFile, binary.LittleEndian, block) == nil {
			for r := uint32(1); r != 4; r++ {
				for k := 0; k < blockSize; k++ {
					block[k] += calcKey(block[(k-1)&(blockSize-1)], block[(k+1)&(blockSize-1)], r*delta, uint32(k))
				}
			}

			// Moving on 1 block back, for record the encoded block
			if _, err := srcFile.Seek(blockSize*-4, 1); err != nil {
				errors <- err
				break
			}

			if err := binary.Write(srcFile, binary.LittleEndian, block); err != nil {
				errors <- err
				break
			}
		}

		path := srcFile.Name()
		srcFile.Close()
		os.Rename(path, path[:len(path)-3]+"lkf")
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

	workerCreator := func(worker func(<-chan *os.File, *sync.WaitGroup, chan<- error)) {
		for n := runtime.NumCPU(); n > 0; n-- {
			wg.Add(1)
			go worker(files, wg, errors)
		}
	}

	log.SetFlags(0)
	switch args[1] {
	case "decode":
		workerCreator(decoder)
		srcExt = ".lkf"
	case "encode":
		workerCreator(encoder)
		srcExt = ".mp3"
	case "version":
		log.Println("LKFCoder version", version)
		return
	default:
		log.Fatal("Specified an unsupported action. Must be decode/encode or version.")
	}

	walker := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || strings.ToLower(filepath.Ext(path)) != srcExt || err != nil {
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

	log.Println("Please wait...")
	start := time.Now()
	for err := range errors {
		log.Println(err)
	}

	log.Printf("Successfully processed %d files in %v\n", counterFiles, time.Since(start))
}
