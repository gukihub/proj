package main

import (
	"fmt"
	"os"
	"os/user"
	"strings"
	"syscall"
)

func ls(dir string) (int, string) {
	var err error
	var fi_buf os.FileInfo
	var fi_tab []os.FileInfo
	var f *os.File

	f, err = os.Open(dir)
	if err != nil {
		return 1, fmt.Sprintf("failed to open %s (%v)\n", dir, err)
	}

	fi_buf, err = f.Stat()
	if err != nil {
		return 1, fmt.Sprintf("failed to stat %s (%v)\n", dir, err)
	}

	if fi_buf.IsDir() == false {
		return 1, fmt.Sprintf("%s is not a directory\n", dir)
	}

	fi_tab, err = f.Readdir(0)
	if err != nil {
		return 1, fmt.Sprintf("failed to read %s (%v)\n", dir, err)
	}

	for _, g := range fi_tab {
		if strings.HasPrefix(g.Name(), ".") {
			continue
		}
		userinfo, _ := user.LookupId(fmt.Sprintf("%d", g.Sys().(*syscall.Stat_t).Uid))
		fmt.Printf("%-11s %s %d %s %s\n",
			g.Mode(),
			userinfo.Username,
			g.Sys().(*syscall.Stat_t).Gid,
			g.ModTime().Format("2006-01-02 15:04:05"),
			g.Name())
	}

	return 0, ""
}
