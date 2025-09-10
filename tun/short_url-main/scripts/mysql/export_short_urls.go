package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 数据库连接
	db, err := sql.Open("mysql", "root:123456@tcp(127.0.0.1:3306)/short_url")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 创建CSV文件
	file, err := os.Create("short_url_all.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV头
	writer.Write([]string{"short_url"})

	// 获取所有short_url_前缀的表
	rows, err := db.Query("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME LIKE 'short_url_%'")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	// 遍历每个表
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			panic(err)
		}

		// 查询该表的short_url
		query := fmt.Sprintf("SELECT short_url FROM %s", tableName)
		dataRows, err := db.Query(query)
		if err != nil {
			panic(err)
		}
		defer dataRows.Close()

		// 写入CSV
		for dataRows.Next() {
			var shortUrl string
			if err := dataRows.Scan(&shortUrl); err != nil {
				panic(err)
			}
			writer.Write([]string{shortUrl})
		}
	}

	fmt.Println("All short_urls exported to short_url_all.csv")
}
