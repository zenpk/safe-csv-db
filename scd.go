package scd

import (
	"encoding/csv"
	"log"
	"os"
	"sync"
)

type RecordType interface {
	// ToRow converts a RecordType instance to []string, this should be defined by user
	ToRow() ([]string, error)
	// FromRow creates a RecordType instance from []string, this should be defined by user
	FromRow(row []string) (RecordType, error)
}

type Table struct {
	recordType RecordType
	rawRecords [][]string
	file       *os.File
	changed    chan struct{}
	close      chan struct{}
	closed     chan error
	mutex      sync.Mutex
}

// OpenTable opens a table (csv file), if not exists then create
func OpenTable(path string, recordType RecordType) (*Table, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(file)
	rawRecords, err := reader.ReadAll()
	if err != nil {
		if err := file.Close(); err != nil {
			log.Fatalln(err)
		}
		return nil, err
	}
	newCsv := &Table{
		recordType: recordType,
		rawRecords: rawRecords,
		file:       file,
		changed:    make(chan struct{}),
		close:      make(chan struct{}),
		closed:     make(chan error),
		mutex:      sync.Mutex{},
	}
	return newCsv, nil
}

// ListenChange listen to recordType change signal, whenever a change happens and the recordType is idle,
// writes the records to the csv file. This function will return an error after the recordType is closed
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
				if err := writer.WriteAll(t.rawRecords); err != nil {
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

// Close the recordType (csv file)
func (t *Table) Close() {
	t.close <- struct{}{}
}

// All returns all rows
func (t *Table) All() ([]RecordType, error) {
	res := make([]RecordType, 0)
	t.mutex.Lock()
	for i := 0; i < len(t.rawRecords); i++ {
		record, err := t.recordType.FromRow(t.rawRecords[i])
		if err != nil {
			return nil, err
		}
		res = append(res, record)
	}
	t.mutex.Unlock()
	return res, nil
}

// Select a row by its id, if there are multiple rows with the same id, the first row will be returned
// The col refers to the index of the id column in the csv file,
// notice that the id must be converted to string in advance. The col starts from 0
func (t *Table) Select(col int, id string) (RecordType, error) {
	t.mutex.Lock()
	for i := 0; i < len(t.rawRecords); i++ {
		if col >= len(t.rawRecords[i]) {
			return nil, FindOutOfIndex
		}
		if t.rawRecords[i][col] == id {
			record, err := t.recordType.FromRow(t.rawRecords[i])
			if err != nil {
				return nil, err
			}
			t.mutex.Unlock()
			return record, nil
		}
	}
	return nil, nil
}

// SelectAll rows that has the specified value on the specified column
func (t *Table) SelectAll(col int, by string) ([]RecordType, error) {
	records := make([]RecordType, 0)
	t.mutex.Lock()
	for i := 0; i < len(t.rawRecords); i++ {
		if col >= len(t.rawRecords[i]) {
			return nil, FindOutOfIndex
		}
		if t.rawRecords[i][col] == by {
			record, err := t.recordType.FromRow(t.rawRecords[i])
			if err != nil {
				return nil, err
			}
			records = append(records, record)
		}
	}
	t.mutex.Unlock()
	return records, nil
}

// Insert a row into the recordType
func (t *Table) Insert(record RecordType) error {
	rawRecord, err := record.ToRow()
	if err != nil {
		return err
	}
	t.mutex.Lock()
	t.rawRecords = append(t.rawRecords, rawRecord)
	t.mutex.Unlock()
	t.changed <- struct{}{}
	return nil
}

// InsertAll rows into the recordType
func (t *Table) InsertAll(records []RecordType) error {
	rawRecords := make([][]string, 0)
	for _, record := range records {
		rawRecord, err := record.ToRow()
		if err != nil {
			return err
		}
		rawRecords = append(rawRecords, rawRecord)
	}
	t.mutex.Lock()
	t.rawRecords = append(t.rawRecords, rawRecords...)
	t.mutex.Unlock()
	t.changed <- struct{}{}
	return nil
}

// Update a row based on its id, col and id work the same as Select
func (t *Table) Update(col int, id string, record RecordType) error {
	rawRecord, err := record.ToRow()
	if err != nil {
		return err
	}
	t.mutex.Lock()
	for i := 0; i < len(t.rawRecords); i++ {
		if col >= len(t.rawRecords[i]) {
			t.mutex.Unlock()
			return FindOutOfIndex
		}
		if t.rawRecords[i][col] == id {
			t.rawRecords[i] = rawRecord
			t.mutex.Unlock()
			t.changed <- struct{}{}
			return nil
		}
	}
	return ValueNotFound
}

// UpdateAll rows that has the specified value on the specified column, col and id work the same as Select
func (t *Table) UpdateAll(col int, by string, record RecordType) error {
	rawRecord, err := record.ToRow()
	if err != nil {
		return err
	}
	updated := false
	t.mutex.Lock()
	for i := 0; i < len(t.rawRecords); i++ {
		if col >= len(t.rawRecords[i]) {
			t.mutex.Unlock()
			return FindOutOfIndex
		}
		if t.rawRecords[i][col] == by {
			t.rawRecords[i] = rawRecord
			updated = true
		}
	}
	t.mutex.Unlock()
	if updated {
		t.changed <- struct{}{}
		return nil
	}
	return ValueNotFound
}

// Delete a row based on its id, col and id work the same as Select
func (t *Table) Delete(col int, id string) error {
	t.mutex.Lock()
	for i := 0; i < len(t.rawRecords); i++ {
		if col >= len(t.rawRecords[i]) {
			return FindOutOfIndex
		}
		if t.rawRecords[i][col] == id {
			t.rawRecords[i] = t.rawRecords[len(t.rawRecords)-1]
			t.rawRecords = t.rawRecords[:len(t.rawRecords)-1]
			t.mutex.Unlock()
			t.changed <- struct{}{}
			return nil
		}
	}
	return ValueNotFound
}

// DeleteAll rows that has the specified value on the specified column, col and id work the same as Select
func (t *Table) DeleteAll(col int, by string) error {
	deleted := false
	t.mutex.Lock()
	for i := len(t.rawRecords) - 1; i >= 0; i-- {
		if col >= len(t.rawRecords[i]) {
			return FindOutOfIndex
		}
		if t.rawRecords[i][col] == by {
			t.rawRecords[i] = t.rawRecords[len(t.rawRecords)-1]
			t.rawRecords = t.rawRecords[:len(t.rawRecords)-1]
			deleted = true
		}
	}
	t.mutex.Unlock()
	if deleted {
		t.changed <- struct{}{}
		return nil
	}
	return ValueNotFound
}
