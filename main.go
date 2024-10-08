package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"hash/fnv"

	"github.com/dolthub/swiss"
)

var CONCURRENCY = runtime.NumCPU() / 2

type ConcurrentHashMap struct {
	sync.RWMutex
	hashmap *swiss.Map[string, string]
}

func (h *ConcurrentHashMap) Set(hash string, path string) {
	h.Lock()
	defer h.Unlock()
	h.hashmap.Put(hash, path)
}

func (h *ConcurrentHashMap) Get(hash string) (string, bool) {
	h.RLock()
	defer h.RUnlock()
	return h.hashmap.Get(hash)
}

func main() {
	// Get the command-line arguments
	args := os.Args

	// Check if the correct number of arguments was provided
	if len(args) < 2 {
		fmt.Println("Usage: go run main.go dir1 dir2 ...")
		return
	}

	hashes := getFiles(args[1:])
	duplicates := checkDupes(hashes)

	var wg sync.WaitGroup
	results := make(chan string, 1000)

	for i := 0; i < CONCURRENCY; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for dupe := range duplicates {
				dupes := strings.Split(dupe, ",")
				sort.Strings(dupes)
				results <- strings.Join(dupes, ", ")
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		fmt.Println(result)
	}
}

func getFiles(dirs []string) <-chan string {
	var wg sync.WaitGroup
	hashes := make(chan string, 1000)
	for _, dir := range dirs {
		wg.Add(1)
		go func(dir string) {
			defer wg.Done()
			err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					fmt.Printf("error accessing path %q: %v\n", path, err)
					return err
				}
				if d.IsDir() {
					return nil
				}
				fileHash, err := getFileHash(path)
				if err != nil {
					fmt.Printf("Error getting hash for file %s: %v\n", path, err)
					return nil
				}
				hashes <- fileHash + ":" + path
				return nil
			})
			if err != nil {
				fmt.Printf("error walking the path %q: %v\n", dir, err)
			}
		}(dir)
	}
	go func() {
		wg.Wait()
		close(hashes)
	}()
	return hashes
}

func checkDupes(fileHashes <-chan string) <-chan string {
	var wg sync.WaitGroup
	duplicates := make(chan string, 1000)
	fileHashToPath := &ConcurrentHashMap{hashmap: swiss.NewMap[string, string](0)}
	for i := 0; i < CONCURRENCY; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for hashString := range fileHashes {
				hashParts := strings.SplitN(hashString, ":", 2)
				if len(hashParts) != 2 {
					fmt.Printf("Invalid hash string: %s\n", hashString)
					continue
				}
				hash := hashParts[0]
				path := hashParts[1]
				if foundPath, exists := fileHashToPath.Get(hash); exists {
					// file has a duplicate hash; verify if same
					if isEqual, err := areFilesEqual(path, foundPath); err == nil && isEqual {
						duplicates <- foundPath + "," + path
					}
				} else {
					// new hash; add to hashmap
					fileHashToPath.Set(hash, path)
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(duplicates)
	}()
	return duplicates
}

func getFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := fnv.New64a()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// areFilesEqual checks if two files have identical contents
func areFilesEqual(path1, path2 string) (bool, error) {
	// First, compare file sizes
	info1, err := os.Stat(path1)
	if err != nil {
		return false, err
	}
	info2, err := os.Stat(path2)
	if err != nil {
		return false, err
	}
	if info1.Size() != info2.Size() {
		return false, nil
	}

	// If sizes are equal, compare contents
	file1, err := os.Open(path1)
	if err != nil {
		return false, err
	}
	defer file1.Close()

	file2, err := os.Open(path2)
	if err != nil {
		return false, err
	}
	defer file2.Close()

	// Compare the contents of the two files byte-by-byte using buffered I/O
	const bufferSize = 64 * 1024 // 64KB buffer
	buf1 := make([]byte, bufferSize)
	buf2 := make([]byte, bufferSize)

	for {
		n1, err1 := file1.Read(buf1)
		n2, err2 := file2.Read(buf2)

		if n1 != n2 || !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false, nil
		}

		if err1 == io.EOF && err2 == io.EOF {
			return true, nil
		}

		if err1 != nil && err1 != io.EOF && err1 != io.ErrUnexpectedEOF {
			return false, err1
		}
		if err2 != nil && err2 != io.EOF && err2 != io.ErrUnexpectedEOF {
			return false, err2
		}
	}
}
