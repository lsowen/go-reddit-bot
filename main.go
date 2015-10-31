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
	Username        string
	Password        string
	AccessToken     string `yaml:"access_token"`
	ClientSecret    string `yaml:"client_secret"`
	Subreddits      []string
	DomainWhitelist []string `yaml:"domain_whitelist"`
}

type Processor struct {
	Config          BotConfig
	Client          *client.Client
	DomainWhitelist map[string]bool
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

		sqlStmt = "create table blocked_domains (domain text not null primary key, blocked_count integer default 1);"
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

func db_mark_blocked_domain(db *sql.DB, domain string) {
	var blocked_count int
	err := db.QueryRow("SELECT blocked_count FROM blocked_domains WHERE domain = ?", domain).Scan(&blocked_count)
	if err == nil {
		blocked_count += 1
		db.Exec("UPDATE blocked_domains SET blocked_count = ? WHERE domain = ?", blocked_count, domain)
	} else {
		db.Exec("INSERT INTO blocked_domains (domain) VALUES (?)", domain)
	}
}

func postEntry(processor Processor, subreddit string, entry client.Listing) error {
	params := client.SubmitLinkParameters{
		Subreddit: subreddit,
		Title:     entry.Data.Title,
		Url:       entry.Data.Url,
	}

	linkResponse, err := processor.Client.SubmitLink(params)

	if err != nil {
		for {
			if captchaError, ok := err.(client.BadCaptchaError); ok {
				fmt.Printf("Bad captcha: https://www.reddit.com/captcha/%s\n", captchaError.CaptchaId)
				fmt.Print("Enter captcha response: ")
				captchaReader := bufio.NewReader(os.Stdin)
				response, _ := captchaReader.ReadString('\n')
				params.CaptchaId = captchaError.CaptchaId
				params.CaptchaResponse = strings.Trim(response, "\n")
				linkResponse, err = processor.Client.SubmitLink(params)
				if err == nil {
					break
				}
			} else {
				return err
			}
		}
	}

	if linkResponse != nil {
		commentTemplate := `Source Post: [reddit.com/%s](https://reddit.com/%s)`
		err = processor.Client.SubmitComment(linkResponse.Name, fmt.Sprintf(commentTemplate, entry.Data.Id, entry.Data.Id))
		if err != nil {
			return err
		}
	}

	return nil
}

func processEntry(processor Processor, entry client.Listing, db *sql.DB) error {
	if entry.Data.Over18 || entry.Data.IsSelf || entry.Data.Score < 5 {
		/* Don't allow NSFW, Self, or low scored items */
		return nil
	}

	if db_entry_exists(db, entry.Data.Id) {
		/* Don't try to repost already posted entry */
		fmt.Printf("Short circuiting %s\n", entry.Data.Id)
		return nil
	}

	if !processor.DomainWhitelist[entry.Data.Domain] {
		db_mark_blocked_domain(db, entry.Data.Domain)
		err := db_entry_record(db, entry.Data.Id)
		return err
	}

	postEntry(processor, "maybemaybeoriginal", entry)
	entry = *(&entry).Copy()
	entry.Data.Title = "Maybe Maybe Maybe"
	postEntry(processor, "maybemaybemaybe", entry)

	err := db_entry_record(db, entry.Data.Id)
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

	processor := Processor{
		Config: config,
		Client: client.New(config.AccessToken, config.ClientSecret),
	}

	processor.Client.Signin(config.Username, config.Password)

	processor.DomainWhitelist = make(map[string]bool, len(processor.Config.DomainWhitelist))
	for _, domain := range processor.Config.DomainWhitelist {
		processor.DomainWhitelist[domain] = true
	}

	db, err := db_init(database_path)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	entries := []client.Listing{}

	for _, subreddit := range processor.Config.Subreddits {
		fmt.Printf("Fetching updates for /r/%s\n", subreddit)
		response, err := processor.Client.GetSubreddit(fmt.Sprintf("/r/%s/hot", subreddit))
		if err != nil {
			fmt.Println(err)
		} else {
			entries = append(entries, response.Data.Children...)
		}
	}

	entryOrder := rand.Perm(len(entries))
	for _, order := range entryOrder {
		entry := entries[order]
		err := processEntry(processor, entry, db)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("Successfully posted %s\n", entry.Data.Permalink)
		}
	}

}
