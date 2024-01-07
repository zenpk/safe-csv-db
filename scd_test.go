package scd

import (
	"errors"
	"strconv"
	"testing"
)

type TestRecordType struct {
	Id   int64
	Name string
}

func (t TestRecordType) ToRow() ([]string, error) {
	row := make([]string, 2)
	row[0] = strconv.FormatInt(t.Id, 10)
	row[1] = t.Name
	return row, nil
}

func (t TestRecordType) FromRow(row []string) (RecordType, error) {
	if len(row) < 2 {
		return nil, errors.New("out of range")
	}
	id, err := strconv.ParseInt(row[0], 10, 64)
	if err != nil {
		return nil, err
	}
	record := TestRecordType{
		Id:   id,
		Name: row[1],
	}
	return record, nil
}

func Test(t *testing.T) {
	table, err := OpenTable("./test.csv", TestRecordType{})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		if err := table.ListenChange(); err != nil {
			panic(err)
		}
	}()

	all, err := table.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	record1 := TestRecordType{
		Id:   1,
		Name: "abc",
	}
	if err := table.Insert(record1); err != nil {
		t.Fatal(err)
	}
	all, err = table.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	record2 := TestRecordType{
		Id:   2,
		Name: "abc",
	}
	if err := table.Insert(record2); err != nil {
		t.Fatal(err)
	}
	all, err = table.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	record3 := TestRecordType{
		Id:   3,
		Name: "def",
	}
	if err := table.Insert(record3); err != nil {
		t.Fatal(err)
	}
	all, err = table.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	select1, err := table.Select(0, "1")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(select1)

	selectAll, err := table.SelectAll(1, "abc")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(selectAll)

	record1.Name = "update abc"
	if err := table.Update(0, "1", record1); err != nil {
		t.Fatal(err)
	}
	all, err = table.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	if err := table.Delete(0, "2"); err != nil {
		t.Fatal(err)
	}
	all, err = table.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	record1.Name = "abc"
	if err := table.InsertAll([]RecordType{record1, record2, record3}); err != nil {
		t.Fatal(err)
	}
	all, err = table.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	record2.Name = "update all abc"
	if err := table.UpdateAll(1, "abc", record2); err != nil {
		t.Fatal(err)
	}
	all, err = table.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	if err := table.DeleteAll(1, "def"); err != nil {
		t.Fatal(err)
	}
	all, err = table.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	table.Close()
}
