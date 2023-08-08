package scd

import (
	"errors"
	"log"
	"testing"
)

type TestTable struct {
	Id   string
	Name string
}

func (t TestTable) ToRow() ([]string, error) {
	row := make([]string, 2)
	row[0] = t.Id
	row[1] = t.Name
	return row, nil
}

func (t TestTable) FromRow(row []string) (Table, error) {
	if len(row) < 2 {
		return nil, errors.New("out of range")
	}
	newTable := TestTable{
		Id:   row[0],
		Name: row[1],
	}
	return newTable, nil
}

func Test(t *testing.T) {
	csv, err := OpenCsv("./test.csv", TestTable{})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		if err := csv.ListenChange(); err != nil {
			log.Fatalln(err)
		}
	}()

	all, err := csv.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	record1 := TestTable{
		Id:   "1",
		Name: "abc",
	}
	if err := csv.Insert(record1); err != nil {
		t.Fatal(err)
	}
	all, err = csv.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	record2 := TestTable{
		Id:   "2",
		Name: "abc",
	}
	if err := csv.Insert(record2); err != nil {
		t.Fatal(err)
	}
	all, err = csv.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	record3 := TestTable{
		Id:   "3",
		Name: "def",
	}
	if err := csv.Insert(record3); err != nil {
		t.Fatal(err)
	}
	all, err = csv.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	select1, err := csv.Select(0, "1")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(select1)

	selectAll, err := csv.SelectAll(1, "abc")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(selectAll)

	record1.Name = "updated abc"
	if err := csv.Update(0, "1", record1); err != nil {
		t.Fatal(err)
	}
	all, err = csv.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	if err := csv.Delete(0, "2"); err != nil {
		t.Fatal(err)
	}
	all, err = csv.All()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(all)

	csv.Close()
}
