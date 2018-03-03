// Utility for encoding/decoding lkf files.
// Used 3-pass block cipher XXTEA with 128-bit key and block size of 128 words.
package main

import (
	"encoding/binary"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	blockSize = 128 // The block size in words. Every word of 32 bit
	delta     = 0x9e3779b9
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
func decode(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info.IsDir() || strings.ToLower(filepath.Ext(path)) != ".lkf" {
		return nil
	}

	srcFile, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		log.Println(path, err)
		return nil
	}
	defer os.Rename(path, path[:len(path)-3]+"mp3")
	defer srcFile.Close()

	var block [blockSize]uint32

	for binary.Read(srcFile, binary.LittleEndian, &block) == nil {
		for r := uint32(3); r != 0; r-- {
			for k := blockSize - 1; k >= 0; k-- {
				block[k] -= calcKey(block[(k-1)&(blockSize-1)], block[(k+1)&(blockSize-1)], r*delta, uint32(k))
			}
		}

		// Moving on 1 block back, for record the decoded block
		if _, err := srcFile.Seek(blockSize*-4, 1); err != nil {
			log.Println(path, err)
			break
		}

		if err := binary.Write(srcFile, binary.LittleEndian, &block); err != nil {
			log.Println(path, err)
			break
		}
	}

	return nil
}

// encode function encrypts the mp3-file and writes the result to the source file, then changes the extension to .lkf
// Encoding occurs by blocks from the beginning of the file. If the end of file is less than block size, it remains as it is.
func encode(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info.IsDir() || strings.ToLower(filepath.Ext(path)) != ".mp3" {
		return nil
	}

	srcFile, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		log.Println(path, err)
		return nil
	}
	defer os.Rename(path, path[:len(path)-3]+"lkf")
	defer srcFile.Close()

	var block [blockSize]uint32

	for binary.Read(srcFile, binary.LittleEndian, &block) == nil {
		for r := uint32(1); r != 4; r++ {
			for k := 0; k < blockSize; k++ {
				block[k] += calcKey(block[(k-1)&(blockSize-1)], block[(k+1)&(blockSize-1)], r*delta, uint32(k))
			}
		}

		// Moving on 1 block back, for record the encoded block
		if _, err := srcFile.Seek(blockSize*-4, 1); err != nil {
			log.Println(path, err)
			break
		}

		if err := binary.Write(srcFile, binary.LittleEndian, &block); err != nil {
			log.Println(path, err)
			break
		}
	}

	return nil
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Not specified action or path")
	}

	var err error
	var action string = os.Args[1] // The desired action: encode or decode
	var path string = os.Args[2]   // Path to the coded file or folder

	startTime := time.Now()
	switch action {
	case "decode":
		err = filepath.Walk(path, decode)
	case "encode":
		err = filepath.Walk(path, encode)
	default:
		err = errors.New("Specified an unsupported action. Must be decode or encode.")
	}
	finishTime := time.Now()

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Operation completed successfully in %v\n", finishTime.Sub(startTime))
}
