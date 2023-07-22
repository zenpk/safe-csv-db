package scd

import "testing"

func Test(t *testing.T) {
	table, err := OpenTable("./temp.csv")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(table.Records)
	if err := table.Insert([]string{"name1", "id1", "1,2,3"}); err != nil {
		t.Fatal(err)
	}
	t.Log(table.Records)
	value, err := table.Select(1, "id1")
	if err != nil {
		t.Fatal(value)
	}
	t.Log(value)
	update := []string{"name2", "id1", "1,2,3"}
	if err := table.Update(1, "id1", update); err != nil {
		t.Fatal(err)
	}
	t.Log(table.Records)
	if err := table.Delete(1, "id1"); err != nil {
		t.Fatal(err)
	}
	t.Log(table.Records)
}