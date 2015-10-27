package main

import (
	"database/sql"
	"fmt"
	"os"
	//"os/user"
	"sort"
	"strings"
	"syscall"
	"time"
)

type dir_t struct {
	name string
	user uint32
	date int64
}

// for sort
type bydate []dir_t

// for sort
func (a bydate) Len() int           { return len(a) }
func (a bydate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a bydate) Less(i, j int) bool { return a[i].date < a[j].date }

func get_dir(dir string) *[]dir_t {
	var err error
	var fi_buf os.FileInfo
	var fi_tab []os.FileInfo
	var f *os.File
	var t []dir_t
	var u dir_t

	t = make([]dir_t, 0)

	f, err = os.Open(dir)
	if err != nil {
		fmt.Printf("failed to open %s (%v)\n", dir, err)
		return nil
	}

	fi_buf, err = f.Stat()
	if err != nil {
		fmt.Printf("failed to stat %s (%v)\n", dir, err)
		return nil
	}

	if fi_buf.IsDir() == false {
		fmt.Printf("%s is not a directory\n", dir)
		return nil
	}

	fi_tab, err = f.Readdir(0)
	if err != nil {
		fmt.Printf("failed to read %s (%v)\n", dir, err)
		return nil
	}

	// loop on the folders
	for _, g := range fi_tab {
		if strings.HasPrefix(g.Name(), ".") {
			continue
		}
		if g.IsDir() == false {
			continue
		}
		u.name = g.Name()
		u.user = g.Sys().(*syscall.Stat_t).Uid
		u.date = g.ModTime().Unix()
		t = append(t, u)
		//fmt.Printf("%s %d %d\n", u.name, u.user, u.date)
	}

	return &t
}

// just for debug
func list_dir_t(d *[]dir_t) {
	for i, u := range *d {
		fmt.Printf("%d %s %d %d\n", i, u.name, u.user, u.date)
	}
	sort.Sort(bydate(*d))
	for i, u := range *d {
		fmt.Printf("%d %s %d %d\n", i, u.name, u.user, u.date)
	}
}

func update_proj(db *sql.DB, d *[]dir_t) {

	sort.Sort(bydate(*d))

	// database statements
	tx, err := db.Begin()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	req := `
	insert into projects(name, author, creation_date)
	values(?, ?, ?)
	`

	stmt, err := tx.Prepare(req)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	// loop on the folders
	for _, u := range *d {
		_, err = stmt.Exec(u.name, u.user,
			time.Unix(u.date, 0).Format("2006-01-02 15:04:05"))
		if err != nil {
			fmt.Printf("error creating project %s (%v)\n",
				u.name, err)
			continue
		}
	}

	stmt.Close()
	tx.Commit()

	return
}
