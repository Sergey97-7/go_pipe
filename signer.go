package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

// сюда писать код
func ExecutePipeline(jobs ...job) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	initCh := make(chan interface{}, 100)
	jobChsSlice := make([]chan interface{}, 0, len(jobs))
	jobChsSlice = append(jobChsSlice, initCh)

	for i, job := range jobs[1:] {
		job1 := job
		i1 := i
		mu.Lock()
		newOutCh := make(chan interface{}, 100)
		jobChsSlice = append(jobChsSlice, newOutCh)
		mu.Unlock()
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			ch := jobChsSlice[i1]
			mu.Unlock()
			job1(ch, newOutCh)
			close(newOutCh)
		}()
	}
	fakeCh := make(chan interface{}, 100)

	jobs[0](fakeCh, jobChsSlice[0])
	close(jobChsSlice[0])
	wg.Wait()
}
func DataSignerCrc32Helper(data string) chan string {
	result := make(chan string, 1)
	go func(out chan<- string) {
		out <- DataSignerCrc32(data)
	}(result)
	return result
}

var md5QuotaCh = make(chan struct{}, 1)

func DataSignerMd5Helper(data string) chan string {
	result := make(chan string, 1)

	md5QuotaCh <- struct{}{}
	firstPart := DataSignerMd5(data)
	<-md5QuotaCh

	secondPart := DataSignerCrc32(firstPart)
	result <- secondPart

	return result
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for i := range in {

		data, ok := i.(int)
		if !ok {
			panic("cant convert result data to string in SingleHash")
		}
		convertedData := strconv.Itoa(data)
		wg.Add(1)
		go func() {
			defer wg.Done()
			var firstPiece string
			var secondPiece string

			ch1 := DataSignerCrc32Helper(convertedData)
			ch2 := DataSignerMd5Helper(convertedData)
			firstPiece = <-ch1
			secondPiece = <-ch2
			out <- firstPiece + "~" + secondPiece
		}()
	}
	wg.Wait()
}
func MultiHash(in, out chan interface{}) {
	const hashPartsNumber = 6
	wg := &sync.WaitGroup{}

	for i := range in {
		data, ok := i.(string)
		if !ok {
			panic("cant convert result data to string in MultiHash")
		}
		mHashSlice := make([]string, hashPartsNumber)
		mu := &sync.Mutex{}

		var count int
		for j := 0; j < hashPartsNumber; j++ {
			j1 := j
			wg.Add(1)
			go func(data string) {
				defer wg.Done()
				hashPartCh := DataSignerCrc32Helper(strconv.Itoa(j1) + data)

				mu.Lock()
				mHashSlice[j1] = <-hashPartCh
				count++
				if count == hashPartsNumber {
					out <- strings.Join(mHashSlice, "")
				}
				mu.Unlock()
			}(data)
		}
	}
	wg.Wait()
}
func CombineResults(in, out chan interface{}) {
	var hashes []string
	for i := range in {
		data, ok := i.(string)
		if !ok {
			panic("cant convert result data to string in CombineResults")
		}
		hashes = append(hashes, data)
	}

	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
	joinedHashes := strings.Join(hashes, "_")
	out <- joinedHashes
}
