package scd

import (
	"encoding/csv"
	"log"
	"os"
	"sync"
)

type Table struct {
	Records [][]string

	file      *os.File
	changed   chan bool
	mutex     sync.Mutex
	waitGroup sync.WaitGroup
}

func NewTable(path string) (*Table, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return tableInit(file, make([][]string, 0))
}

func OpenTable(path string) (*Table, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		file.Close()
		return nil, err
	}
	return tableInit(file, records)
}

func tableInit(file *os.File, records [][]string) (*Table, error) {
	t := &Table{
		Records: records,
		file:    file,
		changed: make(chan bool),
	}
	// whenever changed, write to file
	t.waitGroup.Add(1)
	defer t.waitGroup.Wait()
	go func() {
		for {
			select {
			case changed := <-t.changed:
				log.Printf("%v, waiting", changed)
				writer := csv.NewWriter(t.file)
				t.mutex.Lock()
				writer.Write()

			}
		}
	}()
	return t, nil
}

func (t *Table) Find(pos int, value string) (string, error) {

}

func (t *Table) Insert(value string) {

}
