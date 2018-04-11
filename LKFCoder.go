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
	version   = "0.3" // Program version
)

// The 128-bit key for encrypting/decrypting lkf files. It is divided into 4 parts of 32 bit each.
var key = [4]uint32{
	0x8ac14c27,
	0x42845ac1,
	0x136506bb,
	0x5d47c66,
}

func calcKey(leftWord, rightWord, r, k uint32) uint32 {
	n1 := (leftWord>>5 ^ rightWord<<2) + (rightWord>>3 ^ leftWord<<4)
	n2 := (key[(k&3)^(r>>2&3)] ^ leftWord) + (r ^ rightWord)
	return n1 ^ n2
}

// decode function decrypts the lkf-file and writes the result to the source file, then changes the extension to .mp3
// Decoding occurs by blocks from the beginning of the file. If the end of file is less than block size, it remains as it is.
func decode(srcFile *os.File, wg *sync.WaitGroup, semaphoreCH <-chan struct{}, errorsCH chan<- error) {
	var block [blockSize]uint32
	defer wg.Done()
	defer func() {
		path := srcFile.Name()
		srcFile.Close()
		os.Rename(path, path[:len(path)-3]+"mp3")
		<-semaphoreCH
	}()

	for binary.Read(srcFile, binary.LittleEndian, &block) == nil {
		for r := uint32(3); r != 0; r-- {
			for k := blockSize - 1; k >= 0; k-- {
				block[k] -= calcKey(block[(k-1)&(blockSize-1)], block[(k+1)&(blockSize-1)], r*delta, uint32(k))
			}
		}

		// Moving on 1 block back, for record the decoded block
		if _, err := srcFile.Seek(blockSize*-4, 1); err != nil {
			errorsCH <- err
			break
		}

		if err := binary.Write(srcFile, binary.LittleEndian, &block); err != nil {
			errorsCH <- err
			break
		}
	}
}

// encode function encrypts the mp3-file and writes the result to the source file, then changes the extension to .lkf
// Encoding occurs by blocks from the beginning of the file. If the end of file is less than block size, it remains as it is.
func encode(srcFile *os.File, wg *sync.WaitGroup, semaphoreCH <-chan struct{}, errorsCH chan<- error) {
	var block [blockSize]uint32
	defer wg.Done()
	defer func() {
		path := srcFile.Name()
		srcFile.Close()
		os.Rename(path, path[:len(path)-3]+"lkf")
		<-semaphoreCH
	}()

	for binary.Read(srcFile, binary.LittleEndian, &block) == nil {
		for r := uint32(1); r != 4; r++ {
			for k := 0; k < blockSize; k++ {
				block[k] += calcKey(block[(k-1)&(blockSize-1)], block[(k+1)&(blockSize-1)], r*delta, uint32(k))
			}
		}

		// Moving on 1 block back, for record the encoded block
		if _, err := srcFile.Seek(blockSize*-4, 1); err != nil {
			errorsCH <- err
			break
		}

		if err := binary.Write(srcFile, binary.LittleEndian, &block); err != nil {
			errorsCH <- err
			break
		}
	}
}

func main() {
	args := []string{2: "./"}
	copy(args, os.Args)
	var srcExt string
	var counterFiles int

	var action func(*os.File, *sync.WaitGroup, <-chan struct{}, chan<- error)
	wg := new(sync.WaitGroup)
	errorsCH := make(chan error)
	semaphoreCH := make(chan struct{}, runtime.NumCPU())

	log.SetFlags(0)
	switch args[1] {
	case "decode":
		action = decode
		srcExt = ".lkf"
	case "encode":
		action = encode
		srcExt = ".mp3"
	case "version":
		log.Println("LKFCoder version", version)
		return
	default:
		log.Fatal("Specified an unsupported action. Must be decode/encode or version.")
	}

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || strings.ToLower(filepath.Ext(path)) != srcExt {
			return nil
		}

		srcFile, err := os.OpenFile(path, os.O_RDWR, 0644)
		if err != nil {
			log.Println(path, err)
			return nil
		}

		semaphoreCH <- struct{}{}
		wg.Add(1)
		counterFiles++
		go action(srcFile, wg, semaphoreCH, errorsCH)
		return nil
	}

	log.Println("Please wait...")
	startTime := time.Now()
	if err := filepath.Walk(args[2], walker); err != nil {
		log.Println(err)
	}

	go func() {
		wg.Wait()
		close(errorsCH)
	}()

	for err := range errorsCH {
		log.Println(err)
	}
	finishTime := time.Now()

	log.Printf("Processed files: %d\n", counterFiles)
	log.Printf("Operation completed successfully in %v\n", finishTime.Sub(startTime))
}
