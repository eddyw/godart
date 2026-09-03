// Command sqlmem proves the surface of database/sql with a driver of its own.
// The driver keeps each row in memory, thus the program reaches no server and
// no file. Each result takes an order, thus the output is equal on any machine.
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

// row is 1 row of the table users. The field note holds a null when the
// pointer is nil.
type row struct {
	id   int64
	name string
	age  int64
	note *string
}

// store holds the table. A mutex guards it, because the pool gives 1
// connection to each goroutine at the same time.
type store struct {
	mu   sync.Mutex
	rows []row
	next int64
}

// snapshot copies each row. A transaction takes 1 before a write.
func (s *store) snapshot() []row {
	out := make([]row, len(s.rows))
	copy(out, s.rows)
	return out
}

var db1 = &store{next: 1}

// memDriver is the driver. sql.Register takes it under the name mem.
type memDriver struct{}

func (memDriver) Open(name string) (driver.Conn, error) {
	if name != "db1" {
		return nil, fmt.Errorf("mem: no database %q", name)
	}
	return &memConn{st: db1}, nil
}

// memConn is 1 connection. database/sql gives 1 goroutine to it at a time.
type memConn struct {
	st     *store
	closed bool
	tx     []row
	inTx   bool
}

func (c *memConn) Prepare(query string) (driver.Stmt, error) {
	if c.closed {
		return nil, driver.ErrBadConn
	}
	n, err := inputs(query)
	if err != nil {
		return nil, err
	}
	return &memStmt{c: c, query: query, n: n}, nil
}

func (c *memConn) Close() error {
	c.closed = true
	return nil
}

func (c *memConn) Begin() (driver.Tx, error) {
	c.st.mu.Lock()
	defer c.st.mu.Unlock()
	if c.inTx {
		return nil, errors.New("mem: a transaction already stands")
	}
	c.inTx = true
	c.tx = c.st.snapshot()
	return &memTx{c: c}, nil
}

// QueryContext gives the fast path of database/sql. The package then writes no
// prepared statement.
func (c *memConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if query == "bad" {
		return nil, driver.ErrBadConn
	}
	if query == "slow" {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.query(query, args)
}

// ExecContext gives the fast path of a write.
func (c *memConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.exec(query, args)
}

func (c *memConn) query(query string, args []driver.NamedValue) (driver.Rows, error) {
	c.st.mu.Lock()
	defer c.st.mu.Unlock()
	switch query {
	case "all":
		out := &memRows{cols: []string{"id", "name", "age", "note"}}
		for _, r := range c.st.rows {
			var note driver.Value
			if r.note != nil {
				note = *r.note
			}
			out.data = append(out.data, []driver.Value{r.id, r.name, r.age, note})
		}
		return out, nil
	case "name":
		id, err := argInt(args, "id")
		if err != nil {
			return nil, err
		}
		out := &memRows{cols: []string{"name"}}
		for _, r := range c.st.rows {
			if r.id == id {
				out.data = append(out.data, []driver.Value{r.name})
			}
		}
		return out, nil
	case "count":
		out := &memRows{cols: []string{"count"}}
		out.data = append(out.data, []driver.Value{int64(len(c.st.rows))})
		return out, nil
	case "scan":
		out := &scanRows{}
		out.cols = []string{"id", "name", "age", "note"}
		for _, r := range c.st.rows {
			var note driver.Value
			if r.note != nil {
				note = *r.note
			}
			out.data = append(out.data, []driver.Value{r.id, r.name, r.age, note})
		}
		return out, nil
	}
	return nil, fmt.Errorf("mem: no query %q", query)
}

func (c *memConn) exec(query string, args []driver.NamedValue) (driver.Result, error) {
	c.st.mu.Lock()
	defer c.st.mu.Unlock()
	switch query {
	case "insert":
		if len(args) != 3 {
			return nil, fmt.Errorf("mem: insert takes 3 arguments and got %d", len(args))
		}
		name, ok := args[0].Value.(string)
		if !ok {
			return nil, fmt.Errorf("mem: the name is no string")
		}
		age, ok := args[1].Value.(int64)
		if !ok {
			return nil, fmt.Errorf("mem: the age is no integer")
		}
		var note *string
		if args[2].Value != nil {
			text, ok := args[2].Value.(string)
			if !ok {
				return nil, fmt.Errorf("mem: the note is no string")
			}
			note = &text
		}
		id := c.st.next
		c.st.next++
		c.st.rows = append(c.st.rows, row{id: id, name: name, age: age, note: note})
		return memResult{id: id, n: 1}, nil
	case "delete":
		id, err := argInt(args, "id")
		if err != nil {
			return nil, err
		}
		var kept []row
		var n int64
		for _, r := range c.st.rows {
			if r.id == id {
				n++
				continue
			}
			kept = append(kept, r)
		}
		c.st.rows = kept
		return memResult{id: 0, n: n}, nil
	}
	return nil, fmt.Errorf("mem: no statement %q", query)
}

// argInt reads 1 integer argument. It takes the name of a named argument, and
// the position of an ordinal argument.
func argInt(args []driver.NamedValue, name string) (int64, error) {
	for _, a := range args {
		if a.Name == name || (a.Name == "" && a.Ordinal == 1) {
			v, ok := a.Value.(int64)
			if !ok {
				return 0, fmt.Errorf("mem: the argument %s is no integer", name)
			}
			return v, nil
		}
	}
	return 0, fmt.Errorf("mem: the argument %s is missing", name)
}

func inputs(query string) (int, error) {
	switch query {
	case "all", "count", "scan", "slow", "bad":
		return 0, nil
	case "name", "delete":
		return 1, nil
	case "insert":
		return 3, nil
	}
	return 0, fmt.Errorf("mem: no statement %q", query)
}

// memStmt is a prepared statement.
type memStmt struct {
	c     *memConn
	query string
	n     int
}

func (s *memStmt) Close() error  { return nil }
func (s *memStmt) NumInput() int { return s.n }

func (s *memStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.c.exec(s.query, named(args))
}

func (s *memStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.c.query(s.query, named(args))
}

func named(args []driver.Value) []driver.NamedValue {
	out := make([]driver.NamedValue, len(args))
	for i, a := range args {
		out[i] = driver.NamedValue{Ordinal: i + 1, Value: a}
	}
	return out
}

// memRows gives each row of a result.
type memRows struct {
	cols []string
	data [][]driver.Value
	pos  int
}

func (r *memRows) Columns() []string { return r.cols }
func (r *memRows) Close() error      { return nil }

func (r *memRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

// scanRows gives driver.RowsColumnScanner. database/sql then calls ScanColumn
// in place of Next. The call carries the scan context of the query.
type scanRows struct {
	memRows
	row []driver.Value
}

func (r *scanRows) NextRow() error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	r.row = r.data[r.pos]
	r.pos++
	return nil
}

func (r *scanRows) ScanColumn(scanCtx driver.ScanContext, index int, dest any) error {
	if index >= len(r.row) {
		return fmt.Errorf("mem: no column %d", index)
	}
	return sql.ConvertAssign(scanCtx, dest, r.row[index])
}

// memResult holds the 2 numbers of a write.
type memResult struct {
	id int64
	n  int64
}

func (r memResult) LastInsertId() (int64, error) { return r.id, nil }
func (r memResult) RowsAffected() (int64, error) { return r.n, nil }

// memTx is a transaction. A rollback puts the snapshot back.
type memTx struct{ c *memConn }

func (t *memTx) Commit() error {
	t.c.st.mu.Lock()
	defer t.c.st.mu.Unlock()
	t.c.inTx = false
	t.c.tx = nil
	return nil
}

func (t *memTx) Rollback() error {
	t.c.st.mu.Lock()
	defer t.c.st.mu.Unlock()
	t.c.st.rows = t.c.tx
	t.c.inTx = false
	t.c.tx = nil
	return nil
}

func check(what string, err error) {
	if err != nil {
		fmt.Println(what, "failed:", err)
	}
}

func main() {
	sql.Register("mem", memDriver{})
	fmt.Println("drivers:", sql.Drivers())

	db, err := sql.Open("mem", "db1")
	check("open", err)
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	// The 2 lines start the cleaner goroutine of the pool. The close of the
	// database must stop it, thus the program must end.
	db.SetConnMaxIdleTime(time.Second)
	db.SetConnMaxLifetime(time.Minute)

	check("ping", db.Ping())
	fmt.Println("max open:", db.Stats().MaxOpenConnections)

	// 1. The writes.
	note := "the first"
	res, err := db.Exec("insert", "ada", int64(36), note)
	check("insert", err)
	id, err := res.LastInsertId()
	check("id", err)
	n, err := res.RowsAffected()
	check("rows", err)
	fmt.Println("insert 1:", id, n)

	_, err = db.Exec("insert", "grace", int64(45), nil)
	check("insert", err)
	_, err = db.Exec("insert", "alan", int64(41), "the third")
	check("insert", err)

	// 2. The read of each row.
	rows, err := db.Query("all")
	check("query", err)
	cols, err := rows.Columns()
	check("columns", err)
	fmt.Println("columns:", strings.Join(cols, ","))
	for rows.Next() {
		var id, age int64
		var name string
		var note sql.NullString
		check("scan", rows.Scan(&id, &name, &age, &note))
		fmt.Printf("row: %d %s %d %v %q\n", id, name, age, note.Valid, note.String)
	}
	check("rows", rows.Err())
	check("close", rows.Close())

	// 2b. The read through a driver that scans into the destination itself.
	scan, err := db.Query("scan")
	check("scan query", err)
	for scan.Next() {
		var id, age int64
		var name string
		var note sql.NullString
		check("scan", scan.Scan(&id, &name, &age, &note))
		fmt.Printf("scan: %d %s %d %v\n", id, name, age, note.Valid)
	}
	check("scan rows", scan.Err())
	check("scan close", scan.Close())

	// 3. The read of 1 row.
	var name string
	err = db.QueryRow("name", int64(2)).Scan(&name)
	check("row", err)
	fmt.Println("name of 2:", name)

	err = db.QueryRow("name", int64(99)).Scan(&name)
	fmt.Println("no rows:", errors.Is(err, sql.ErrNoRows))

	// 4. The prepared statement.
	stmt, err := db.Prepare("name")
	check("prepare", err)
	var got string
	check("stmt scan", stmt.QueryRow(int64(3)).Scan(&got))
	fmt.Println("prepared:", got)
	check("stmt close", stmt.Close())

	// 5. The transaction that commits.
	tx, err := db.Begin()
	check("begin", err)
	_, err = tx.Exec("insert", "edsger", int64(72), nil)
	check("tx insert", err)
	check("commit", tx.Commit())
	fmt.Println("after commit:", count(db))
	fmt.Println("commit again:", errors.Is(tx.Commit(), sql.ErrTxDone))

	// 6. The transaction that rolls back.
	tx2, err := db.Begin()
	check("begin", err)
	_, err = tx2.Exec("insert", "john", int64(53), nil)
	check("tx insert", err)
	var inTx int64
	check("tx count", tx2.QueryRow("count").Scan(&inTx))
	fmt.Println("inside the transaction:", inTx)
	check("rollback", tx2.Rollback())
	fmt.Println("after the rollback:", count(db))

	// 7. The delete.
	res, err = db.Exec("delete", int64(4))
	check("delete", err)
	n, err = res.RowsAffected()
	check("rows", err)
	fmt.Println("delete:", n, count(db))

	// 8. A context that a cancel already stopped.
	dead, stop := context.WithCancel(context.Background())
	stop()
	_, err = db.QueryContext(dead, "all")
	fmt.Println("canceled:", errors.Is(err, context.Canceled))

	// 9. A context with a deadline, against a query that waits.
	slow, stopSlow := context.WithTimeout(context.Background(), 20*time.Millisecond)
	_, err = db.QueryContext(slow, "slow")
	fmt.Println("deadline:", errors.Is(err, context.DeadlineExceeded))
	stopSlow()

	// 10. The pool under load. 8 goroutines share 2 connections, thus each one
	// waits for a free connection.
	out := make(chan string, 8)
	var wg sync.WaitGroup
	for i := 1; i <= 8; i++ {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			var name string
			id := int64(k%3 + 1)
			if err := db.QueryRow("name", id).Scan(&name); err != nil {
				out <- fmt.Sprintf("worker %d: %v", k, err)
				return
			}
			out <- fmt.Sprintf("worker %d: %s", k, name)
		}(i)
	}
	wg.Wait()
	close(out)
	var lines []string
	for line := range out {
		lines = append(lines, line)
	}
	sort.Strings(lines)
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println("max open after the load:", db.Stats().MaxOpenConnections)

	// 11. A dedicated connection.
	conn, err := db.Conn(context.Background())
	check("conn", err)
	var one int64
	check("conn count", conn.QueryRowContext(context.Background(), "count").Scan(&one))
	fmt.Println("count of the connection:", one)
	check("conn close", conn.Close())

	// 12. The pool wakes a goroutine that waits. 2 dedicated connections hold
	// the whole pool, thus the query of the goroutine finds no connection. The
	// close of the first connection must wake it.
	c1, err := db.Conn(context.Background())
	check("conn 1", err)
	c2, err := db.Conn(context.Background())
	check("conn 2", err)
	done := make(chan string, 1)
	go func() {
		var name string
		if err := db.QueryRow("name", int64(1)).Scan(&name); err != nil {
			done <- fmt.Sprint("waiter: ", err)
			return
		}
		done <- "waiter: " + name
	}()
	// The sleep gives the goroutine the time to reach the wait of the pool.
	time.Sleep(20 * time.Millisecond)
	check("close 1", c1.Close())
	fmt.Println(<-done)
	check("close 2", c2.Close())

	// 13. The opener goroutine of the pool. 2 dedicated connections fill the
	// pool again. The bad connection then leaves the pool. The opener
	// goroutine must open a connection for the goroutine that waits.
	h1, err := db.Conn(context.Background())
	check("hold 1", err)
	h2, err := db.Conn(context.Background())
	check("hold 2", err)
	opened := make(chan string, 1)
	go func() {
		var n int64
		if err := db.QueryRow("count").Scan(&n); err != nil {
			opened <- fmt.Sprint("opener: ", err)
			return
		}
		opened <- fmt.Sprint("opener: ", n)
	}()
	// The sleep gives the goroutine the time to reach the wait of the pool.
	time.Sleep(20 * time.Millisecond)
	_, err = h1.QueryContext(context.Background(), "bad")
	fmt.Println("the bad connection:", err)
	fmt.Println(<-opened)
	fmt.Println("open connections:", db.Stats().OpenConnections)
	check("hold 2 close", h2.Close())

	// 14. The close, and a use after it.
	check("db close", db.Close())
	err = db.Ping()
	fmt.Println("after the close:", err)
}

func count(db *sql.DB) int64 {
	var n int64
	if err := db.QueryRow("count").Scan(&n); err != nil {
		fmt.Println("count failed:", err)
		return -1
	}
	return n
}
