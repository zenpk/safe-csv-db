# safe-csv-db

Thread safe database using only CSV

<https://pkg.go.dev/github.com/zenpk/safe-csv-db>

## Functions

This package supports these functions

- `All`: SELECT *
- `Select`: SELECT * BY LIMIT 1
- `SelectAll`: SELECT * BY
- `Insert`: Insert one row
- `InsertAll`: Insert multiple rows
- `Update`: Update one row by some value
- `UpdateAll`: Update all rows by some value
- `Delete`: Delete one row by some value
- `DeleteAll`: Delete all rows by some value

It is recommended that every row has a unique id

All values are stored as `string`

For detailed API please refer to the go doc

## Usage

### Basic

```go
// define your RecordType struct
type My struct{
    Id int64
    Name string
}

// implement the RecordType interface
func (m My) ToRow() ([]string, error) {
    row := make([]string, 2)
    row[0] = strconv.FormatInt(m.Id, 10)
    row[1] = m.Name
    return row, nil
}

func (m My) FromRow(row []string) (RecordType, error) {
    if len(row) < 2 {
        return nil, errors.New("out of range")
    }
    id, err := strconv.ParseInt(row[0], 10, 64)
    if err != nil {
        return nil, err
    }
    record := My{
        Id:   id,
        Name: row[1],
    }
    return record, nil
}

tableMy, err := OpenTable("./my.csv", My{})
defer tableMy.Close()

go func() {
    if err := tableMy.ListenChange(); err != nil {
        panic(err)
    }
}()

record := My{
    Id:   1,
    Name: "abc",
}
if err := tableMy.Insert(record); err != nil {
    panic(err)
}
```

### Call from other functions

```go
func InitDb(preparing chan<- struct{}, stop <-chan struct{}) {
    tableUser, err := OpenTable("./users.csv", User{})
    defer tableUser.Close()

    go func() {
        if err := tableUser.ListenChange(); err != nil {
            panic(err)
        }
    }()

    tableArticle, err := OpenTable("./articles.csv", Article{})
    defer tableArticle.Close()

    go func() {
        if err := tableArticle.ListenChange(); err != nil {
            panic(err)
        }
    }()

    close(preparing)
    <- stop
}

func main() {
    preparing := make(chan struct{})
    stop := make(chan struct{})
    go InitDb(preparing, stop)
    <- preparing
    // do something
    close(stop)
}
```
