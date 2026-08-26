package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
)

type config struct {
	addr, db  string
	selfcheck bool
}

func parseConfig() (config, error) {
	var c config
	flag.StringVar(&c.addr, "addr", "127.0.0.1:19081", "HTTP 监听地址")
	flag.StringVar(&c.db, "db", "bladeready.db", "SQLite 数据库路径")
	flag.BoolVar(&c.selfcheck, "selfcheck", false, "执行完整业务自检后退出")
	flag.Parse()
	addrSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "addr" {
			addrSet = true
		}
	})
	if port := os.Getenv("PORT"); port != "" && !addrSet {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return c, fmt.Errorf("PORT 必须是 1-65535 的端口号")
		}
		c.addr = net.JoinHostPort("127.0.0.1", port)
	}
	host, port, err := net.SplitHostPort(c.addr)
	if err != nil {
		return c, fmt.Errorf("-addr 格式无效: %w", err)
	}
	if host == "" || port == "" {
		return c, fmt.Errorf("-addr 必须包含主机和端口")
	}
	return c, nil
}
