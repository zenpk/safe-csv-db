# safe-csv-db

Thread safe database using only CSV

## Functions

This package only supports `Select`, `SelectAll`, `Insert`, `Update` and `Delete`

Every row should have a unique id

All values are stored as `string`

For detailed API please refer to the go doc

## Usage

### Basic

```go
table, err := OpenTable("./test.csv")
defer table.Close()

go func() {
    if err := table.ListenChange(); err != nil {
        log.fatalln(err)
    }
}()

table.Insert([]string{"a", "b", "c"})
```

### Call by other functions

```go
func InitDb(ready, done chan struct{}) {
    table1, err := OpenTable("./first.csv")
    defer table1.Close()
    
    go func() {
        if err := table1.ListenChange(); err != nil {
            log.fatalln(err)
        }
    }()
    
    table2, err := OpenTable("./second.csv")
    defer table2.Close()
    
    go func() {
        if err := table2.ListenChange(); err != nil {
            log.fatalln(err)
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
