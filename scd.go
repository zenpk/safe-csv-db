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
	close     chan bool
	closed    chan error
	mutex     sync.Mutex
	waitGroup sync.WaitGroup
}

func OpenTable(path string) (*Table, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		if err := file.Close(); err != nil {
			log.Fatalln(err)
		}
		return nil, err
	}
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
			case <-t.changed:
				writer := csv.NewWriter(t.file)
				t.mutex.Lock()
				if err := writer.WriteAll(records); err != nil {
					log.Fatalln(err)
				}
				t.mutex.Unlock()
			case <-t.close:
				err := t.file.Close()
				t.closed <- err
				return
			}
		}
	}()
	return t, nil
}

func (t *Table) Close() error {
	t.close <- true
	for {
		select {
		case err := <-t.closed:
			return err
		}
	}
}

func (t *Table) Select(col int, id string) ([]string, error) {
	for i := 0; i < len(t.Records); i++ {
		if col >= len(t.Records[i]) {
			return make([]string, 0), FindOutOfIndex
		}
		if t.Records[i][col] == id {
			return t.Records[i], nil
		}
	}
	return make([]string, 0), nil
}

func (t *Table) SelectAll(col int, value string) ([][]string, error) {
	res := make([][]string, 0)
	for i := 0; i < len(t.Records); i++ {
		if col >= len(t.Records[i]) {
			return make([][]string, 0), FindOutOfIndex
		}
		if t.Records[i][col] == value {
			res = append(res, t.Records[i])
		}
	}
	return res, nil
}

func (t *Table) Insert(value []string) error {
	t.mutex.Lock()
	t.Records = append(t.Records, value)
	t.mutex.Unlock()
	return nil
}

func (t *Table) Update(col int, id string, values []string) error {
	t.mutex.Lock()
	row, err := t.find(col, id)
	if err != nil {
		t.mutex.Unlock()
		return err
	}
	if row < 0 {
		return ValueNotFound
	}
	t.Records[row] = values
	t.mutex.Unlock()
	return nil
}

func (t *Table) Delete(col int, id string) error {
	for i := 0; i < len(t.Records); i++ {
		if col >= len(t.Records[i]) {
			return FindOutOfIndex
		}
		if t.Records[i][col] == id {
			t.mutex.Lock()
			t.Records[i] = t.Records[len(t.Records)-1]
			t.Records = t.Records[:len(t.Records)-1]
			t.mutex.Unlock()
			return nil
		}
	}
	return ValueNotFound
}

func (t *Table) find(col int, id string) (int, error) {
	for i := 0; i < len(t.Records); i++ {
		if col >= len(t.Records[i]) {
			return -1, FindOutOfIndex
		}
		if t.Records[i][col] == id {
			return i, nil
		}
	}
	return -1, nil
}
