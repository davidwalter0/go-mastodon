package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mattn/go-mastodon"
	"github.com/mattn/go-tty"
	"golang.org/x/net/html"
)

var (
	toot = flag.String("t", "", "toot text")
)

func extractText(node *html.Node, w *bytes.Buffer) {
	if node.Type == html.TextNode {
		data := strings.Trim(node.Data, "\r\n")
		if data != "" {
			w.WriteString(data)
		}
	} else if node.Type == html.ElementNode {
		if node.Data == "li" {
			w.WriteString("\n* ")
		}
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, w)
	}
}

func prompt() (string, string, error) {
	t, err := tty.Open()
	if err != nil {
		return "", "", err
	}
	defer t.Close()

	fmt.Print("E-Mail: ")
	b, _, err := bufio.NewReader(os.Stdin).ReadLine()
	if err != nil {
		return "", "", err
	}
	email := string(b)

	fmt.Print("Password: ")
	password, err := t.ReadPassword()
	if err != nil {
		return "", "", err
	}
	return email, password, nil
}

func getConfig() (string, *mastodon.Config, error) {
	dir := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		dir = os.Getenv("APPDATA")
		if dir == "" {
			dir = filepath.Join(os.Getenv("USERPROFILE"), "Application Data", "mstdn")
		}
		dir = filepath.Join(dir, "mstdn")
	} else {
		dir = filepath.Join(dir, ".config", "mstdn")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", nil, err
	}
	file := filepath.Join(dir, "settings.json")
	b, err := ioutil.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return "", nil, err
	}
	config := &mastodon.Config{
		Server:       "https://mstdn.jp",
		ClientID:     "7d1873f3940af3e9128c81d5a2ddb3f235ccfa1cd11761efd3b8426f40898fe8",
		ClientSecret: "3c8ea997c580f196453e97c1c58f6f5c131f668456bbe1ed37aaccac719397db",
	}
	if err == nil {
		err = json.Unmarshal(b, &config)
		if err != nil {
			return "", nil, fmt.Errorf("could not unmarshal %v: %v", file, err)
		}
	}
	return file, config, nil
}

func main() {
	flag.Parse()

	file, config, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := mastodon.NewClient(config)

	if config.AccessToken == "" {
		email, password, err := prompt()
		if err != nil {
			log.Fatal(err)
		}
		err = client.Authenticate(email, password)
		if err != nil {
			log.Fatal(err)
		}
		b, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			log.Fatal("failed to store file:", err)
		}
		err = ioutil.WriteFile(file, b, 0700)
		if err != nil {
			log.Fatal("failed to store file:", err)
		}
		return
	}

	if *toot != "" {
		_, err = client.PostStatus(&mastodon.Toot{
			Status: *toot,
		})
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	timeline, err := client.GetTimelineHome()
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range timeline {
		doc, err := html.Parse(strings.NewReader(t.Content))
		if err != nil {
			log.Fatal(err)
		}
		var buf bytes.Buffer
		extractText(doc, &buf)
		fmt.Println(t.Account.Username)
		fmt.Println(buf.String())
	}
}
