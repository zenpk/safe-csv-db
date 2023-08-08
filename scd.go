package scd

import (
	"encoding/csv"
	"log"
	"os"
	"sync"
)

type Table interface {
	// ToRow converts a Table instance to []string, this should be defined by user
	ToRow() ([]string, error)
	// FromRow creates a Table instance from []string, this should be defined by user
	FromRow(row []string) (Table, error)
}

type Csv struct {
	table      Table
	rawRecords [][]string
	file       *os.File
	changed    chan struct{}
	close      chan struct{}
	closed     chan error
	mutex      sync.Mutex
}

// OpenCsv opens a table (csv file), if not exists then create
func OpenCsv(path string, table Table) (*Csv, error) {
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
	newCsv := &Csv{
		table:      table,
		rawRecords: rawRecords,
		file:       file,
		changed:    make(chan struct{}),
		close:      make(chan struct{}),
		closed:     make(chan error),
		mutex:      sync.Mutex{},
	}
	return newCsv, nil
}

// ListenChange listen to table change signal, whenever a change happens and the table is idle,
// writes the records to the csv file. This function will return an error after the table is closed
func (c *Csv) ListenChange() error {
	go func() {
		for {
			select {
			case <-c.changed:
				writer := csv.NewWriter(c.file)
				c.mutex.Lock()
				if err := c.file.Truncate(0); err != nil {
					log.Fatalln(err)
				}
				if _, err := c.file.Seek(0, 0); err != nil {
					log.Fatalln(err)
				}
				if err := writer.WriteAll(c.rawRecords); err != nil {
					log.Fatalln(err)
				}
				c.mutex.Unlock()
			case <-c.close:
				err := c.file.Close()
				c.closed <- err
				return
			}
		}
	}()
	err := <-c.closed
	return err
}

// Close the table (csv file)
func (c *Csv) Close() {
	c.close <- struct{}{}
}

// All returns all rows
func (c *Csv) All() ([]Table, error) {
	res := make([]Table, 0)
	c.mutex.Lock()
	for i := 0; i < len(c.rawRecords); i++ {
		table, err := c.table.FromRow(c.rawRecords[i])
		if err != nil {
			return nil, err
		}
		res = append(res, table)
	}
	c.mutex.Unlock()
	return res, nil
}

// Select a row by its id, if there are multiple rows with the same id, the first row will be returned
// The col refers to the index of the id column in the csv file,
// notice that the id must be converted to string in advance. The col starts from 0
func (c *Csv) Select(col int, id string) (Table, error) {
	c.mutex.Lock()
	for i := 0; i < len(c.rawRecords); i++ {
		if col >= len(c.rawRecords[i]) {
			return nil, FindOutOfIndex
		}
		if c.rawRecords[i][col] == id {
			table, err := c.table.FromRow(c.rawRecords[i])
			if err != nil {
				return nil, err
			}
			c.mutex.Unlock()
			return table, nil
		}
	}
	return nil, nil
}

// SelectAll rows that has the specified value on the specified column
func (c *Csv) SelectAll(col int, by string) ([]Table, error) {
	res := make([]Table, 0)
	c.mutex.Lock()
	for i := 0; i < len(c.rawRecords); i++ {
		if col >= len(c.rawRecords[i]) {
			return nil, FindOutOfIndex
		}
		if c.rawRecords[i][col] == by {
			table, err := c.table.FromRow(c.rawRecords[i])
			if err != nil {
				return nil, err
			}
			res = append(res, table)
		}
	}
	c.mutex.Unlock()
	return res, nil
}

// Insert a row into the table
func (c *Csv) Insert(value Table) error {
	rawRecord, err := value.ToRow()
	if err != nil {
		return err
	}
	c.mutex.Lock()
	c.rawRecords = append(c.rawRecords, rawRecord)
	c.mutex.Unlock()
	c.changed <- struct{}{}
	return nil
}

// InsertAll rows into the table
func (c *Csv) InsertAll(values []Table) error {
	rawRecords := make([][]string, 0)
	for _, value := range values {
		rawRecord, err := value.ToRow()
		if err != nil {
			return err
		}
		rawRecords = append(rawRecords, rawRecord)
	}
	c.mutex.Lock()
	c.rawRecords = append(c.rawRecords, rawRecords...)
	c.mutex.Unlock()
	c.changed <- struct{}{}
	return nil
}

// Update a row based on its id, col and id work the same as Select
func (c *Csv) Update(col int, id string, value Table) error {
	rawRecord, err := value.ToRow()
	if err != nil {
		return err
	}
	c.mutex.Lock()
	for i := 0; i < len(c.rawRecords); i++ {
		if col >= len(c.rawRecords[i]) {
			c.mutex.Unlock()
			return FindOutOfIndex
		}
		if c.rawRecords[i][col] == id {
			c.rawRecords[i] = rawRecord
			c.mutex.Unlock()
			c.changed <- struct{}{}
			return nil
		}
	}
	return ValueNotFound
}

// UpdateAll rows that has the specified value on the specified column, col and id work the same as Select
func (c *Csv) UpdateAll(col int, by string, value Table) error {
	rawRecord, err := value.ToRow()
	if err != nil {
		return err
	}
	updated := false
	c.mutex.Lock()
	for i := 0; i < len(c.rawRecords); i++ {
		if col >= len(c.rawRecords[i]) {
			c.mutex.Unlock()
			return FindOutOfIndex
		}
		if c.rawRecords[i][col] == by {
			c.rawRecords[i] = rawRecord
			updated = true
		}
	}
	c.mutex.Unlock()
	if updated {
		c.changed <- struct{}{}
		return nil
	}
	return ValueNotFound
}

// Delete a row based on its id, col and id work the same as Select
func (c *Csv) Delete(col int, id string) error {
	c.mutex.Lock()
	for i := 0; i < len(c.rawRecords); i++ {
		if col >= len(c.rawRecords[i]) {
			return FindOutOfIndex
		}
		if c.rawRecords[i][col] == id {
			c.rawRecords[i] = c.rawRecords[len(c.rawRecords)-1]
			c.rawRecords = c.rawRecords[:len(c.rawRecords)-1]
			c.mutex.Unlock()
			c.changed <- struct{}{}
			return nil
		}
	}
	return ValueNotFound
}

// DeleteAll rows that has the specified value on the specified column, col and id work the same as Select
func (c *Csv) DeleteAll(col int, by string) error {
	deleted := false
	c.mutex.Lock()
	for i := len(c.rawRecords) - 1; i >= 0; i-- {
		if col >= len(c.rawRecords[i]) {
			return FindOutOfIndex
		}
		if c.rawRecords[i][col] == by {
			c.rawRecords[i] = c.rawRecords[len(c.rawRecords)-1]
			c.rawRecords = c.rawRecords[:len(c.rawRecords)-1]
			deleted = true
		}
	}
	c.mutex.Unlock()
	if deleted {
		c.changed <- struct{}{}
		return nil
	}
	return ValueNotFound
}
