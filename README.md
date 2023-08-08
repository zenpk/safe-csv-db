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

myCsv, err := OpenCsv("./test.csv", My{})
defer myCsv.Close()

go func() {
    if err := myCsv.ListenChange(); err != nil {
        log.Fatalln(err)
    }
}()

record := My{
    Id:   1,
    Name: "abc",
}
if err := csv.Insert(record); err != nil {
    log.Fatalln(err)
}
```

### Call from other functions

```go
func InitDb(ready, done chan struct{}) {
    userCsv, err := OpenCsv("./users.csv", User{})
    defer userCsv.Close()

    go func() {
        if err := userCsv.ListenChange(); err != nil {
            log.Fatalln(err)
        }
    }()

    articleCsv, err := OpenCsv("./articles.csv", Article{})
    defer articleCsv.Close()

    go func() {
        if err := articleCsv.ListenChange(); err != nil {
            log.Fatalln(err)
        }
    }()

    ready <- struct{}{}
    <- done
}

func main() {
    ready := make(chan struct{})
    done := make(chan struct{})
    go InitDb(done, ready)
    <- ready
    // do something
    done <- struct{}{}
}
```
