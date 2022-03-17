package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"golang.org/x/sync/semaphore"
)

type Thief struct {
	destination string
	event       int
	mu          sync.Mutex
	imageList   map[string]string
}

const (
	Limit  = 100
	Weight = 1
)

func main() {
	thief := Thief{
		destination: "./files",
		event:       1491,
		imageList:   make(map[string]string),
	}

	err := os.MkdirAll(thief.destination, 0777)
	if err != nil {
		log.Fatal("error making directory", err)
	}

	maxGroups := 350
	for i := 1; i <= 9999; i++ {
		file := fmt.Sprintf("DSC_%04d.JPG", i)
		thief.imageList[file] = ""
	}
	sem := semaphore.NewWeighted(Limit)
	var wg sync.WaitGroup
	limit := make(chan int, 10)
	defer close(limit)

	for i := 1; i <= maxGroups; i++ {
		wg.Add(1)
		i := i
		go func() {
			sem.Acquire(context.Background(), Weight)
			defer wg.Done()
			thief.getPhotos(i)
			sem.Release(Weight)
		}()
	}

	wg.Wait()

}

func (t *Thief) getPhotos(group int) {
	for file, image := range t.imageList {
		if image == "" {
			url := fmt.Sprintf("https://d1ahne26oxheso.cloudfront.net/preview/2022/Event%d/%d/%s", t.event, group, file)

			fmt.Printf("Checking URL: %s\n", url)

			resp, err := http.Get(url)
			if err != nil {
				log.Println(err)
				continue
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				continue
			}
			t.mu.Lock()
			defer t.mu.Unlock()

			err = os.MkdirAll(fmt.Sprintf("%s/%d", t.destination, group), 0777)
			if err != nil {
				log.Fatal("error making directory", err)
			}

			dest, err := os.Create(fmt.Sprintf("%s/%d/%s", t.destination, group, file))
			if err != nil {
				log.Fatal(err)
			}
			defer dest.Close()

			size, err := io.Copy(dest, resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			t.imageList[file] = strconv.Itoa(group)

			fmt.Printf("Downloaded file: %s size: %d\n", image, size)
		}
	}
}
