package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"gitlab.com/toby3d/telegraph"
	"gopkg.in/ini.v1"
)

// Define variables

var articleID = 0

// A structure to store Telegraph and bot tokens
type cfg struct {
	telegraphToken string
	botToken       string
	chatIDInt      int64
}

// A map to store articles
var articles = make(map[int][]map[string]string)

var (
	page    *telegraph.Page
	content []telegraph.Node
)

func main() {
	inidata, err := ini.Load("rsa-parser.conf") // Loading config file
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}
	section := inidata.Section("tg-variables")

	// Retrieving data from config file
	parserVars := cfg{
		telegraphToken: section.Key("telegraph-token").String(),
		botToken:       section.Key("bot-token").String(),
		chatIDInt:      section.Key("chat-id").MustInt64(0),
	}

	// Starting parser's instance
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"https://ultimasnoticias.com.ve/"},
		ParseFunc: parseArticle,
	}).Start()

	// Creating Telegram Bot with debug on
	// Note: Please keep in mind that default logger may expose sensitive information, use in development only
	bot, err := telego.NewBot(parserVars.botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Get chat ID from the message
	chatID := tu.ID(parserVars.chatIDInt)                                                                                                                                                       // Getting chat ID from struct
	rndID := rand.Intn(articleID)                                                                                                                                                               // Choosing a pseudo-random article to show
	content, _ = telegraph.ContentFormat("<h4><i>" + articles[rndID][0]["description"] + "</i></h4><figure><img src=" + articles[rndID][0]["img"] + "/></figure>" + articles[rndID][0]["text"]) // Building a content for Telepraph page

	// Connecting to Telegraph account
	account := telegraph.Account{
		AccessToken: parserVars.telegraphToken,
		ShortName:   "RSA-bot",
		AuthorName:  "Random Spanish Article bot",
	}

	// Building a Telegraph page
	pageData := telegraph.Page{
		Title:       articles[rndID][0]["title"],
		Description: articles[rndID][0]["description"],
		AuthorName:  account.AuthorName,
		Content:     content,
	}

	// Creating a page that we described above
	page, _ = account.CreatePage(pageData, false)

	// Sending message to chat
	bot.SendMessage(tu.Message(
		chatID,
		page.Title+"\n\n"+articles[rndID][0]["description"]+"\n\n"+page.URL,
	))

	// A temporary debug information
	fmt.Println(articles[rndID][0]["title"] + articles[rndID][0]["description"] + articles[rndID][0]["img"])
}

// Function that checks does a string has at least one of the multiple substrings
func checkSubstrings(str string, subs ...string) bool {
	isMatch := false

	for _, sub := range subs {
		if strings.Contains(str, sub) {
			isMatch = true
		}
	}
	return isMatch
}

// The main parsing function
func parseArticle(g *geziyor.Geziyor, r *client.Response) {
	r.HTMLDoc.Find("div.td_module_flex.td_module_flex_1.td_module_wrap.td-animation-stack.td-cpt-post").Each(func(i int, s *goquery.Selection) { // Looking for an article preview pattern
		var articleDate = s.Find("time.entry-date.updated.td-module-date").Text() // Retrieving article's date

		if articleDate != "" {

			var title = s.Find("h3.entry-title.td-module-title").Text() // Retrieving article's title
			var articleText string
			var articleP []string

			if link, ok := s.Find("a").Attr("href"); ok { // Looking for a link to the full article
				g.Get(r.JoinURL(link), func(_g *geziyor.Geziyor, _r *client.Response) { // Parsing an article's page
					_r.HTMLDoc.Find("body").Find("p").Each(func(_ int, sel *goquery.Selection) { // Retrieving article's text

						// Cutting the useless tail below the article's text
						if !checkSubstrings(sel.Text(), "Guardar mi nombre", "medio impreso y digital", "Wordle lleno de", "mensaje a La Voz del Lector") {
							articleP = append(articleP, sel.Text())
							articleText = strings.Join(articleP, "<p>")
						}
					})

					var description string
					var articleImage string
					_r.HTMLDoc.Find("meta").Each(func(i int, s *goquery.Selection) {
						if name, _ := s.Attr("name"); name == "twitter:description" {
							description, _ = s.Attr("content") // Retrieving article's description
						}
						if name, _ := s.Attr("name"); name == "twitter:image" {
							articleImage, _ = s.Attr("content") // Retrieving article's image
						}
					})
					if articleText != "" {
						tmpMap := map[string]string{ // Stuffing the map of articles
							"title":       title,
							"link":        link,
							"date":        articleDate,
							"text":        articleText,
							"description": description,
							"img":         articleImage,
							"id":          strconv.Itoa(articleID),
						}
						articles[articleID] = append(articles[articleID], tmpMap)
						articleID++
					}
				})
			}
		}
	})
}
