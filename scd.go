package scd

import (
	"encoding/csv"
	"log"
	"os"
	"sync"
)

type Table struct {
	Records [][]string

	file    *os.File
	changed chan struct{}
	close   chan struct{}
	closed  chan error
	mutex   sync.Mutex
}

// OpenTable opens a table (csv file), if not exists then create
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
		changed: make(chan struct{}),
		close:   make(chan struct{}),
		closed:  make(chan error),
		mutex:   sync.Mutex{},
	}
	return t, nil
}

// ListenChange listen to table change signal, whenever a change happens and the table is idle,
// writes the records to the csv file. This function will return an error after the table is closed
func (t *Table) ListenChange() error {
	go func() {
		for {
			select {
			case <-t.changed:
				writer := csv.NewWriter(t.file)
				t.mutex.Lock()
				if err := t.file.Truncate(0); err != nil {
					log.Fatalln(err)
				}
				if _, err := t.file.Seek(0, 0); err != nil {
					log.Fatalln(err)
				}
				if err := writer.WriteAll(t.Records); err != nil {
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
	err := <-t.closed
	return err
}

// Close the table
func (t *Table) Close() {
	t.close <- struct{}{}
}

// Select a row by its id. The col refers to the index of the id column in the csv file,
// starts from 0
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

// SelectAll rows that has the specified value on the specified column
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

// Insert a row to the table
func (t *Table) Insert(value []string) error {
	t.mutex.Lock()
	t.Records = append(t.Records, value)
	t.mutex.Unlock()
	t.changed <- struct{}{}
	return nil
}

// Update a row based on its id, col and id work the same as Select
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
	t.changed <- struct{}{}
	return nil
}

// Delete a row based on its id, col and id work the same as Select
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
			t.changed <- struct{}{}
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
