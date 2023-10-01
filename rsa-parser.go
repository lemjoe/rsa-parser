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

var articleID = 0

type cfg struct {
	telegraphToken string
	botToken       string
	chatIDInt      int64
}

var articles = make(map[int][]map[string]string)

var (
	page    *telegraph.Page
	content []telegraph.Node
)

func main() {
	inidata, err := ini.Load("rsa-parser.conf")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}
	section := inidata.Section("tg-variables")

	parserVars := cfg{
		telegraphToken: section.Key("telegraph-token").String(),
		botToken:       section.Key("bot-token").String(),
		chatIDInt:      section.Key("chat-id").MustInt64(0),
	}

	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"https://ultimasnoticias.com.ve/"},
		ParseFunc: parseArticle,
	}).Start()

	// Create Bot with debug on
	// Note: Please keep in mind that default logger may expose sensitive information, use in development only
	bot, err := telego.NewBot(parserVars.botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Get chat ID from the message
	chatID := tu.ID(parserVars.chatIDInt)
	rndID := rand.Intn(articleID)
	content, _ = telegraph.ContentFormat("<h4><i>" + articles[rndID][0]["description"] + "</i></h4><figure><img src=" + articles[rndID][0]["img"] + "/></figure>" + articles[rndID][0]["text"])

	// Create new Telegraph account.
	account := telegraph.Account{
		AccessToken: parserVars.telegraphToken,
		ShortName:   "RSA-bot", // required
		// Author name/link can be epmty. So secure. Much anonymously. Wow.
		AuthorName: "Random Spanish Article bot", // optional
	}

	pageData := telegraph.Page{
		Title:       articles[rndID][0]["title"],
		Description: articles[rndID][0]["description"],
		AuthorName:  account.AuthorName,
		Content:     content,
	}

	page, _ = account.CreatePage(pageData, false)

	bot.SendMessage(tu.Message(
		chatID,
		page.Title+"\n\n"+articles[rndID][0]["description"]+"\n\n"+page.URL,
	))

	fmt.Println(articles[rndID][0]["title"] + articles[rndID][0]["description"] + articles[rndID][0]["img"])
}

func checkSubstrings(str string, subs ...string) bool {

	isMatch := false

	for _, sub := range subs {
		if strings.Contains(str, sub) {
			isMatch = true
		}
	}
	return isMatch
}

func parseArticle(g *geziyor.Geziyor, r *client.Response) {
	r.HTMLDoc.Find("div.td_module_flex.td_module_flex_1.td_module_wrap.td-animation-stack.td-cpt-post").Each(func(i int, s *goquery.Selection) {
		var articleDate = s.Find("time.entry-date.updated.td-module-date").Text()

		if articleDate != "" {

			var title = s.Find("h3.entry-title.td-module-title").Text()
			var articleText string
			var articleP []string

			if link, ok := s.Find("a").Attr("href"); ok {
				g.Get(r.JoinURL(link), func(_g *geziyor.Geziyor, _r *client.Response) {
					_r.HTMLDoc.Find("body").Find("p").Each(func(_ int, sel *goquery.Selection) {
						if !checkSubstrings(sel.Text(), "Guardar mi nombre", "medio impreso y digital", "Wordle lleno de", "mensaje a La Voz del Lector") {
							articleP = append(articleP, sel.Text())
							articleText = strings.Join(articleP, "<p>")
						}
					})

					var description string
					var articleImage string
					_r.HTMLDoc.Find("meta").Each(func(i int, s *goquery.Selection) {
						if name, _ := s.Attr("name"); name == "twitter:description" {
							description, _ = s.Attr("content")
						}
						if name, _ := s.Attr("name"); name == "twitter:image" {
							articleImage, _ = s.Attr("content")
						}
					})
					if articleText != "" {
						tmpMap := map[string]string{
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
