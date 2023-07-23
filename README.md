# safe-csv-db

Thread safe database using only CSV

## Functions

This package only supports `Select`, `SelectAll`, `Insert`, `Update` and `Delete`

Every row should have a unique id

All values are stored as `string`

For detailed API please refer to the go doc

## Usage

```go
table, err := OpenTable("./test.csv")
defer table.Close()

go func () {
    if err := table.ListenChange(); err != nil {
        log.fatalln(err)
    }
}()

table.Insert([]string{"a", "b", "c"})
```
