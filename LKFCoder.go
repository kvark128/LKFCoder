// Command-line utility for encoding/decoding LKF files.
package main

import (
	"flag"
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

const UsageString = `Использование: %v [опции] <команда> [путь...]

Опции:
 -v    Включает подробный вывод журнала работы программы.
 -w=<число>    Задаёт число горутин-воркеров.
  По умолчанию число горутин равно числу доступных в системе логических процессоров.

Команда:
 decode    Декодирование lkf-файлов в mp3
 encode    Кодирование mp3-файлов в lkf

Путь: Один или более путей к обрабатываемым файлам или каталогам.
 Требуемые файлы определяются по расширению имени файла. *.lkf при декодировании и *.mp3 при кодировании.
 Если в качестве пути указан каталог, то поиск нужных файлов будет выполнен рекурсивно во всех вложенных подкаталогах.
 Если ни один путь не указан, то для поиска файлов будет использоваться текущий рабочий каталог.
`

type CryptorFunc func(*lkf.Cryptor, []byte) int

func FileCryptor(path string, cryptor CryptorFunc) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}

	buf := make([]byte, lkf.BlockSize*1024) // 512 Kb
	c := new(lkf.Cryptor)
	var off int64

	for err == nil {
		var n int
		n, err = io.ReadFull(f, buf)
		if err != nil {
			if err != io.ErrUnexpectedEOF {
				// Fatal error when reading from file or end of file without data
				// Processing must be aborted immediately
				break
			}
			// End of file, but there is read data
			// We try to process them, and then break with io.EOF
			err = io.EOF
		}

		// Encrypt or decrypt the read data
		// Processed data should be written back to the file instead of the previously read ones
		if np := cryptor(c, buf[:n]); np != 0 {
			if _, wErr := f.WriteAt(buf[:np], off); wErr != nil {
				// Fatal error when writing to file
				// Processing must be aborted immediately
				err = wErr
				break
			}
			off += int64(np)
		}
	}

	if err != io.EOF {
		// Fatal error occurred while processing the file. Close the file and return this error
		f.Close()
		return err
	}

	// Just the end of the file with io.EOF. Closing it
	return f.Close()
}

func worker(pathCH <-chan string, wg *sync.WaitGroup, logger *log.Logger, targetExt string, cryptor CryptorFunc) {
	defer wg.Done()
	for path := range pathCH {
		targetPath := strings.TrimSuffix(path, filepath.Ext(path)) + targetExt
		tmpPath := targetPath + ".tmp"
		if err := os.Rename(path, tmpPath); err != nil {
			logger.Printf("worker: %v\n", err)
			continue
		}

		if err := FileCryptor(tmpPath, cryptor); err != nil {
			logger.Printf("worker: %v\n", err)
			continue
		}

		if err := os.Rename(tmpPath, targetPath); err != nil {
			logger.Printf("worker: %v\n", err)
			continue
		}
	}
}

func main() {
	var fileCounter int
	var srcExt, targetExt string
	var cryptor CryptorFunc
	wg := new(sync.WaitGroup)
	pathCH := make(chan string)
	logger := log.New(os.Stdout, "", 0)

	var verbosityFlag bool
	var numWorkersFlag int
	flag.BoolVar(&verbosityFlag, "v", false, "")
	flag.IntVar(&numWorkersFlag, "w", runtime.NumCPU(), "")
	flag.Usage = func() {
		logger.Printf(UsageString, os.Args[0])
	}
	flag.Parse()

	if numWorkersFlag <= 0 {
		logger.Fatalf("No available workers\n")
	}

	cmd := flag.Arg(0)
	switch cmd {
	case "decode":
		cryptor = func(c *lkf.Cryptor, data []byte) int { return c.Decrypt(data, data) }
		srcExt = ".lkf"
		targetExt = ".mp3"
	case "encode":
		cryptor = func(c *lkf.Cryptor, data []byte) int { return c.Encrypt(data, data) }
		srcExt = ".mp3"
		targetExt = ".lkf"
	default:
		logger.Fatalf("Unsupported command specified\n")
	}

	// The first argument is the command. Paths are all arguments after the command.
	// Note that if the user did not specify a command, then the next line will cause a panic!
	paths := flag.Args()[1:]

	if len(paths) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			logger.Fatalf("Unable to get current working directory: %v\n", err)
		}
		paths = append(paths, wd)
	}

	for n := 0; n < numWorkersFlag; n++ {
		wg.Add(1)
		go worker(pathCH, wg, logger, targetExt, cryptor)
	}

	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.ToLower(filepath.Ext(path)) == srcExt {
			fileCounter++
			pathCH <- path
		}
		return nil
	}

	start := time.Now()
	if verbosityFlag {
		logger.Printf("Start processing with %d workers\n", numWorkersFlag)
	}

	for _, path := range paths {
		if verbosityFlag {
			logger.Printf("Walking by path: %v\n", path)
		}
		if err := filepath.WalkDir(path, walker); err != nil {
			logger.Printf("Filewalker: %v\n", err)
			break
		}
	}

	close(pathCH)
	wg.Wait()

	finish := time.Since(start)
	if verbosityFlag {
		logger.Printf("Processed %d *%s files in %v\n", fileCounter, srcExt, finish)
	}
}
