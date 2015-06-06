package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math/rand"
	"maybemaybemaybe_bot/client"
	"os"
	"strings"
)

type BotConfig struct {
	Username     string
	Password     string
	AccessToken  string `yaml:"access_token"`
	ClientSecret string `yaml:"client_secret"`
	Subreddits   []string
}

func db_init(databasePath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(databasePath); os.IsNotExist(err) {
		sqlStmt := "create table seen_entries (id text not null primary key);"
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

func db_entry_exists(db *sql.DB, id string) bool {
	var found_id string
	err := db.QueryRow("SELECT * FROM seen_entries WHERE id = ?", id).Scan(&found_id)
	if err == nil {
		return true
	}

	return false
}

func db_entry_record(db *sql.DB, id string) error {
	_, err := db.Exec("INSERT INTO seen_entries (id) VALUES (?)", id)
	return err
}

func processEntry(c *client.Client, entry client.Listing, db *sql.DB) error {
	if entry.Data.Over18 || entry.Data.IsSelf {
		/* Don't allow NSFW posts or Self posts */
		return nil
	}

	if db_entry_exists(db, entry.Data.Id) {
		/* Don't try to repost already posted entry */
		fmt.Printf("Short circuiting %s\n", entry.Data.Id)
		return nil
	}

	params := client.SubmitLinkParameters{
		Subreddit: "maybemaybemaybe",
		Title:     entry.Data.Title,
		Url:       entry.Data.Url,
	}
	linkResponse, err := c.SubmitLink(params)
	if err != nil {
		for {
			if captchaError, ok := err.(client.BadCaptchaError); ok {
				fmt.Printf("Bad captcha: https://www.reddit.com/captcha/%s\n", captchaError.CaptchaId)
				fmt.Print("Enter captcha response: ")
				captchaReader := bufio.NewReader(os.Stdin)
				response, _ := captchaReader.ReadString('\n')
				params.CaptchaId = captchaError.CaptchaId
				params.CaptchaResponse = strings.Trim(response, "\n")
				linkResponse, err = c.SubmitLink(params)
				if err == nil {
					break
				}
			} else {
				return err
			}
		}
	}

	if linkResponse != nil {
		commentTemplate := `Original Post: [reddit.com/%s](//reddit.com/%s)`
		err = c.SubmitComment(linkResponse.Name, fmt.Sprintf(commentTemplate, entry.Data.Id, entry.Data.Id))
		if err != nil {
			return err
		}
	}

	err = db_entry_record(db, entry.Data.Id)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var database_path string
	var config_path string
	var config BotConfig

	flag.StringVar(&database_path, "db", "bot.db", "Path to bot sqlite3 database file")
	flag.StringVar(&config_path, "config", "bot.yml", "Path to bot yaml config file")
	flag.Parse()

	config_data, err := ioutil.ReadFile(config_path)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = yaml.Unmarshal(config_data, &config)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(config)

	c := client.New(config.AccessToken, config.ClientSecret)
	c.Signin(config.Username, config.Password)

	db, err := db_init(database_path)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	entries := []client.Listing{}

	for _, subreddit := range config.Subreddits {
		fmt.Printf("Fetching updates for /r/%s\n", subreddit)
		response, err := c.GetSubreddit(fmt.Sprintf("/r/%s/hot", subreddit))
		if err != nil {
			fmt.Println(err)
		} else {
			entries = append(entries, response.Data.Children...)
		}
	}

	entryOrder := rand.Perm(len(entries))
	for _, order := range entryOrder {
		entry := entries[order]
		err := processEntry(c, entry, db)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("Successfully posted %s\n", entry.Data.Permalink)
		}
	}

}
