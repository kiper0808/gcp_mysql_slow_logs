package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/heroku/x/hmetrics/onload"
)

const (
	user     = "root"
	password = "root"
	host     = "127.0.0.1:3306"
	dbName   = "slow_queries"
)

func main() { // ips
	bytes, err := os.ReadFile("mysqlslow.json")
	if err != nil {
		panic(fmt.Errorf("file open err: %v", err))
	}
	regex := *regexp.MustCompile(`\# Time: (\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{1,6}Z)\\n\# User@Host: watchdocs\[watchdocs\] @  \[cloudsqlproxy~(\d{1,3}.\d{1,3}.\d{1,3}.\d{1,3})\]. thread_id: (\d{1,})  server_id: (\d{1,})\\n# Query_time: (\d{1,}.\d{1,})  Lock_time: (\d{1,}.\d{1,}) Rows_sent: (\d{1,}). Rows_examined: (\d{1,})\\nSET timestamp=(\d{1,});\\n(.*?);`)
	res := regex.FindAllStringSubmatch(string(bytes), -1)

	db, err := dbConnection()
	if err != nil {
		panic(fmt.Errorf("open db error: %v", err))
	}
	for i := range res { // 2023-03-26T07:05:01.843333Z
		tm, err := time.Parse("2006-01-02T15:04:05.000000Z", res[i][1])
		if err != nil {
			panic(fmt.Errorf("time parse err: %v", err))
		}
		threadId, err := strconv.Atoi(res[i][3])
		if err != nil {
			panic(fmt.Errorf("threadId parse err: %v", err))
		}
		serverId, err := strconv.Atoi(res[i][4])
		if err != nil {
			panic(fmt.Errorf("serverId parse err: %v", err))
		}

		queryTime, err := strconv.ParseFloat(res[i][5], 32)
		if err != nil {
			panic(fmt.Errorf("queryTime parse err: %v", err))
		}

		lockTime, err := strconv.ParseFloat(res[i][6], 32)
		if err != nil {
			panic(fmt.Errorf("serverId parse err: %v", err))
		}

		rowsSent, err := strconv.Atoi(res[i][7])
		if err != nil {
			panic(fmt.Errorf("rows sent parse err: %v", err))
		}

		rowsExamined, err := strconv.Atoi(res[i][8])
		if err != nil {
			panic(fmt.Errorf("rows examined parse err: %v", err))
		}

		googleTimestamp, err := strconv.Atoi(res[i][9])
		if err != nil {
			panic(fmt.Errorf("google timestamp parse err: %v", err))
		}

		fmt.Println(res[i][10])

		slowLog := Log{
			Time:          tm,
			CloudSqlProxy: res[i][2],
			ThreadId:      threadId,
			ServerId:      serverId,
			QueryTime:     queryTime,
			LockTime:      lockTime,
			RowsSent:      rowsSent,
			RowsExamined:  rowsExamined,
			TimeStamp:     googleTimestamp,
			Query:         res[i][10],
		}

		err = insertLogRow(db, slowLog)
		if err != nil {
			panic(fmt.Errorf("insert err: %v", err))
		}
	}
	defer db.Close()
}

func dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s", user, password, host, dbName)
}

func dbConnection() (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn())
	if err != nil {
		log.Printf("Error %s when opening DB\n", err)
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(time.Minute * 5)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		log.Printf("Errors %s pinging DB", err)
		return nil, err
	}
	log.Printf("Connected to DB %s successfully\n", dbName)
	return db, nil
}

type Log struct {
	Time          time.Time
	CloudSqlProxy string
	ThreadId      int
	ServerId      int
	QueryTime     float64
	LockTime      float64
	RowsSent      int
	RowsExamined  int
	TimeStamp     int
	Query         string
}

func insertLogRow(db *sql.DB, l Log) error {
	log.Println(l)
	query := "INSERT INTO slow_queries(google_time, sql_proxy, thread_id, server_id, query_time, lock_time, rows_sent, rows_examined, google_timestamp, sql_query) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, l.Time, l.CloudSqlProxy, l.ThreadId, l.ServerId, l.QueryTime, l.LockTime, l.RowsSent, l.RowsExamined, l.TimeStamp, l.Query)
	if err != nil {
		log.Printf("Error %s when inserting row into table", err)
		return err
	}
	_, err = res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	return nil
}
