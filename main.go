package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

var CONCURRENCY = runtime.NumCPU() / 2

type ConcurrentHashMap struct {
	sync.Mutex
	hashmap map[string]string
}

func (h *ConcurrentHashMap) Set(hash string, path string) {
	h.Lock()
	defer h.Unlock()
	h.hashmap[hash] = path
}

func (h *ConcurrentHashMap) Get(hash string) (string, error) {
	h.Lock()
	defer h.Unlock()
	if path, exists := h.hashmap[hash]; exists {
		return path, nil
	} else {
		return "", fmt.Errorf("%s not found", path)
	}
}

func main() {
	// Get the command-line arguments
	args := os.Args

	// Check if the correct number of arguments was provided
	if len(args) < 2 {
		fmt.Println("Usage: go run main.go dir1 dir2 ...")
		return
	}

	hashes := getFiles(args)
	duplicates := checkDupes(hashes)

	for dupe := range duplicates {
		// sort & format the output
		dupes := strings.Split(dupe, ",")
		sort.Sort(sort.StringSlice(dupes))
		fmt.Printf("%s\n", strings.Join(dupes, ", "))
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
					fmt.Printf("Error getting hash for file:", path, err)
				}
				hashes <- fileHash + ":" + path
				return nil
			})
			if err != nil {
				fmt.Printf("error walking the path %q: %v\n", dir, err)
				return
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
	duplicates := make(chan string, 10)
	fileHashToPath := ConcurrentHashMap{hashmap: make(map[string]string)}
	for i := 0; i < CONCURRENCY; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for hashString := range fileHashes {
				hashParts := strings.Split(hashString, ":")
				hash := hashParts[0]
				path := hashParts[1]
				if foundPath, err := fileHashToPath.Get(hash); err == nil {
					// file has a duplicate hash; verify if same
					if isEqual, err := areFilesEqual(path, foundPath); isEqual && err == nil {
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
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash), nil
}

// areFilesEqual checks if two files have identical contents
func areFilesEqual(path1, path2 string) (bool, error) {
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
	reader1 := bufio.NewReader(file1)
	reader2 := bufio.NewReader(file2)

	for {
		byte1, err1 := reader1.ReadByte()
		byte2, err2 := reader2.ReadByte()

		if err1 == nil && err2 == nil {
			if byte1 != byte2 {
				return false, nil
			}
		}
		if err1 == io.EOF && err2 != io.EOF {
			return false, nil
		} else if err2 == io.EOF && err1 != io.EOF {
			return false, nil
		} else if err1 == io.EOF && err2 == io.EOF {
			return true, nil
		}
	}
}
