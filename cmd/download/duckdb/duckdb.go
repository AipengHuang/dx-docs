package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

// 预下载 Excel 解析所需扩展，避免运行时依赖外网。
var duckdbExtensions = []string{"spatial", "excel"}

func downloadExtensions() {
	ctx := context.Background()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	for _, extension := range duckdbExtensions {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("INSTALL %s;", extension)); err != nil {
			panic(fmt.Errorf("failed to install %s extension: %w", extension, err))
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("LOAD %s;", extension)); err != nil {
			panic(fmt.Errorf("failed to load %s extension: %w", extension, err))
		}
	}
}

func main() {
	downloadExtensions()
}
