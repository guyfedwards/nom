package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Item struct {
	ID          int
	Author      string
	Title       string
	FeedURL     string
	FeedName    string // added from config if set
	Link        string
	Content     string
	ReadAt      time.Time
	PublishedAt time.Time
	UpdatedAt   time.Time
	CreatedAt   time.Time
}

func (i Item) Read() bool {
	return !i.ReadAt.IsZero()
}

type Store interface {
	UpsertItem(item Item) error
	GetAllItems() ([]Item, error)
	GetAllFeedURLs() ([]string, error)
	ToggleRead(ID int) error
	DeleteByFeedURL(feedurl string) error
}

type SQLiteStore struct {
	path string
	db   *sql.DB
}

func NewSQLiteStore(basePath string) (*SQLiteStore, error) {
	dbpath := filepath.Join(basePath, "nom.db")

	info, _ := os.Stat(dbpath)

	db, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		return nil, fmt.Errorf("NewSQLiteCache: %w", err)
	}

	// if there was no db file before we create the connection then we want to run
	// the initial queries now that sqlite db has been created and connected to
	if info == nil {
		err = dbSetup(db)
		if err != nil {
			return nil, fmt.Errorf("NewSQLiteCache: %w", err)
		}
	}

	err = runMigrations(db)
	if err != nil {
		return nil, fmt.Errorf("NewSQLiteCache: %w", err)
	}

	return &SQLiteStore{
		path: dbpath,
		db:   db,
	}, nil
}

// dbSetup runs the initial db queries to create tables etc
func dbSetup(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("dbSetup: %w", err)
	}

	stm := `
		create table items (id integer primary key, feedurl text, link text, title text, content text, author text, readat datetime, publishedat datetime, updatedat datetime, createdat datetime);
		create table migrations (id integer not null, runat datetime);
	`

	_, err = tx.Exec(stm)
	if err != nil {
		return fmt.Errorf("sqlite.go: could not execute query: %w, %s", err, stm)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("sqlite.go: could not commit tx: %w", err)
	}

	return nil
}

func runMigrations(db *sql.DB) (err error) {
	getCurrent := `select count(*) from migrations;`

	migrations := []string{}

	tx, _ := db.Begin()
	updateMigrations, _ := tx.Prepare(`insert into migrations (id, runat) values (?, ?);`)

	var count int
	err = tx.QueryRow(getCurrent).Scan(&count)
	if err != nil {
		return fmt.Errorf("[store.go] runMigrations: %w", err)
	}

	for i, m := range migrations {
		// if the migration has already been run, skip
		if i < count {
			continue
		}

		_, err = tx.Exec(m)
		if err != nil {
			break
		}

		_, err = updateMigrations.Exec(i, time.Now())
		if err != nil {
			break
		}
	}

	if err != nil {
		err = tx.Rollback()
	} else {
		err = tx.Commit()
	}

	return err
}

func (sls SQLiteStore) UpsertItem(item Item) error {
	stmt, err := sls.db.Prepare(`select count(id), id from items where feedurl = ? and title = ?;`)
	if err != nil {
		return fmt.Errorf("sqlite.go: could not prepare query: %w", err)
	}

	var count int
	var id sql.NullInt32
	err = stmt.QueryRow(item.FeedURL, item.Title).Scan(&count, &id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("store.go: write %w", err)
	}

	if count == 0 {
		stmt, err = sls.db.Prepare(`insert into items (feedurl, link, title, content, author, publishedat, createdat, updatedat) values (?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("sqlite.go: could not prepare query: %w", err)
		}

		_, err = stmt.Exec(item.FeedURL, item.Link, item.Title, item.Content, item.Author, item.PublishedAt, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("sqlite.go: Upsert failed: %w", err)
		}
	} else {
		stmt, err = sls.db.Prepare(`update items set content = ?, updatedat = ? where id = ?`)
		if err != nil {
			return fmt.Errorf("sqlite.go: could not prepare query: %w", err)
		}

		_, err = stmt.Exec(item.Content, time.Now(), id)
		if err != nil {
			return fmt.Errorf("sqlite.go: Upsert failed: %w", err)
		}
	}

	return nil
}

// TODO: pagination
func (sls SQLiteStore) GetAllItems() ([]Item, error) {
	stmt := `
		select id, feedurl, link, title, content, author, readat, publishedat, createdat, updatedat from items order by coalesce(publishedat, createdat);
	`

	rows, err := sls.db.Query(stmt)
	if err != nil {
		return []Item{}, fmt.Errorf("store.go: GetAllItems: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		var readAtNull sql.NullTime
		var publishedAtNull sql.NullTime
		var linkNull sql.NullString

		if err := rows.Scan(&item.ID, &item.FeedURL, &linkNull, &item.Title, &item.Content, &item.Author, &readAtNull, &publishedAtNull, &item.CreatedAt, &item.UpdatedAt); err != nil {
			fmt.Println("errrerre: ", err)
			continue
		}

		item.Link = linkNull.String
		item.ReadAt = readAtNull.Time
		item.PublishedAt = publishedAtNull.Time

		items = append(items, item)
	}

	return items, nil
}

func (sls SQLiteStore) ToggleRead(ID int) error {
	stmt, _ := sls.db.Prepare(`update items set readat = ? where id = ?`)

	_, err := stmt.Exec(time.Now(), ID)
	if err != nil {
		return fmt.Errorf("[store.go] ToggleRead: %w", err)
	}

	return nil
}

func (sls SQLiteStore) GetAllFeedURLs() ([]string, error) {
	var urls []string

	stmt, _ := sls.db.Prepare(`select feedurl from items group by feedurl;`)

	rows, err := stmt.Query()
	if err != nil {
		return urls, fmt.Errorf("[store.go] GetAllFeedURLs: %w", err)
	}

	for rows.Next() {
		var feedurl string

		err := rows.Scan(&feedurl)
		if err != nil {
			return urls, fmt.Errorf("[store.go] GetAllFeedURLs: %w", err)
		}

		urls = append(urls, feedurl)
	}

	return urls, nil
}

func (sls SQLiteStore) DeleteByFeedURL(feedurl string) error {
	stmt, _ := sls.db.Prepare(`delete from items where feedurl = ?`)

	_, err := stmt.Exec(feedurl)
	if err != nil {
		return fmt.Errorf("[store.go] DeleteByFeedURL: %w", err)
	}

	return nil
}
