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
	rows       [][]string
	file       *os.File
	changed    chan struct{}
	close      chan struct{}
	mutex      sync.Mutex
}

// OpenTable opens a table (csv file), if not exists then create
func OpenTable(path string, recordType RecordType) (*Table, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		if err := file.Close(); err != nil {
			log.Fatalln(err)
		}
		return nil, err
	}
	newCsv := &Table{
		recordType: recordType,
		rows:       rows,
		file:       file,
		changed:    make(chan struct{}, 1),
		close:      make(chan struct{}),
		mutex:      sync.Mutex{},
	}
	return newCsv, nil
}

// ListenChange listen to recordType change signal, whenever a change happens and the recordType is idle,
// writes the records to the csv file. This function will return an error after the recordType is closed
func (t *Table) ListenChange() error {
	for {
		select {
		case <-t.changed:
			writer := csv.NewWriter(t.file)
			t.mutex.Lock()
			if err := t.file.Truncate(0); err != nil {
				panic(err)
			}
			if _, err := t.file.Seek(0, 0); err != nil {
				panic(err)
			}
			if err := writer.WriteAll(t.rows); err != nil {
				panic(err)
			}
			t.mutex.Unlock()
		case <-t.close:
			return t.file.Close()
		}
	}
}

// Close the recordType (csv file)
func (t *Table) Close() {
	close(t.close)
}

// All returns all rows
func (t *Table) All() ([]RecordType, error) {
	res := make([]RecordType, 0)
	t.mutex.Lock()
	defer t.mutex.Unlock()
	for i := 0; i < len(t.rows); i++ {
		record, err := t.recordType.FromRow(t.rows[i])
		if err != nil {
			return nil, err
		}
		res = append(res, record)
	}
	return res, nil
}

// Select a row by its id, if there are multiple rows with the same id, the first row will be returned
// The col refers to the index of the id column in the csv file,
// notice that the id must be converted to string in advance. The col starts from 0
func (t *Table) Select(col int, id string) (RecordType, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	for i := 0; i < len(t.rows); i++ {
		if col >= len(t.rows[i]) {
			return nil, FindOutOfIndex
		}
		if t.rows[i][col] == id {
			record, err := t.recordType.FromRow(t.rows[i])
			if err != nil {
				return nil, err
			}
			return record, nil
		}
	}
	return nil, nil
}

// SelectAll rows that has the specified value on the specified column
func (t *Table) SelectAll(col int, by string) ([]RecordType, error) {
	records := make([]RecordType, 0)
	t.mutex.Lock()
	defer t.mutex.Unlock()
	for i := 0; i < len(t.rows); i++ {
		if col >= len(t.rows[i]) {
			return nil, FindOutOfIndex
		}
		if t.rows[i][col] == by {
			record, err := t.recordType.FromRow(t.rows[i])
			if err != nil {
				return nil, err
			}
			records = append(records, record)
		}
	}
	return records, nil
}

// Insert a row into the recordType
func (t *Table) Insert(record RecordType) error {
	row, err := record.ToRow()
	if err != nil {
		return err
	}
	t.mutex.Lock()
	t.rows = append(t.rows, row)
	t.mutex.Unlock()
	// use select to avoid channel block
	select {
	case t.changed <- struct{}{}:
	default:
	}
	return nil
}

// InsertAll rows into the recordType
func (t *Table) InsertAll(records []RecordType) error {
	rows := make([][]string, 0)
	for _, record := range records {
		row, err := record.ToRow()
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}
	t.mutex.Lock()
	t.rows = append(t.rows, rows...)
	t.mutex.Unlock()
	// use select to avoid channel block
	select {
	case t.changed <- struct{}{}:
	default:
	}
	return nil
}

// Update a row based on its id, col and id work the same as Select
func (t *Table) Update(col int, id string, record RecordType) error {
	row, err := record.ToRow()
	if err != nil {
		return err
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	for i := 0; i < len(t.rows); i++ {
		if col >= len(t.rows[i]) {
			return FindOutOfIndex
		}
		if t.rows[i][col] == id {
			t.rows[i] = row
			// use select to avoid channel block
			select {
			case t.changed <- struct{}{}:
			default:
			}
			return nil
		}
	}
	return ValueNotFound
}

// UpdateAll rows that has the specified value on the specified column, col and id work the same as Select
func (t *Table) UpdateAll(col int, by string, record RecordType) error {
	row, err := record.ToRow()
	if err != nil {
		return err
	}
	updated := false
	t.mutex.Lock()
	defer t.mutex.Unlock()
	for i := 0; i < len(t.rows); i++ {
		if col >= len(t.rows[i]) {
			return FindOutOfIndex
		}
		if t.rows[i][col] == by {
			t.rows[i] = row
			updated = true
		}
	}
	if updated {
		// use select to avoid channel block
		select {
		case t.changed <- struct{}{}:
		default:
		}
		return nil
	}
	return ValueNotFound
}

// Delete a row based on its id, col and id work the same as Select
func (t *Table) Delete(col int, id string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	for i := 0; i < len(t.rows); i++ {
		if col >= len(t.rows[i]) {
			return FindOutOfIndex
		}
		if t.rows[i][col] == id {
			t.rows[i] = t.rows[len(t.rows)-1]
			t.rows = t.rows[:len(t.rows)-1]
			// use select to avoid channel block
			select {
			case t.changed <- struct{}{}:
			default:
			}
			return nil
		}
	}
	return ValueNotFound
}

// DeleteAll rows that has the specified value on the specified column, col and id work the same as Select
func (t *Table) DeleteAll(col int, by string) error {
	deleted := false
	t.mutex.Lock()
	defer t.mutex.Unlock()
	for i := len(t.rows) - 1; i >= 0; i-- {
		if col >= len(t.rows[i]) {
			return FindOutOfIndex
		}
		if t.rows[i][col] == by {
			t.rows[i] = t.rows[len(t.rows)-1]
			t.rows = t.rows[:len(t.rows)-1]
			deleted = true
		}
	}
	if deleted {
		// use select to avoid channel block
		select {
		case t.changed <- struct{}{}:
		default:
		}
		return nil
	}
	return ValueNotFound
}
