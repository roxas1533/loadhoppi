package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/Songmu/go-httpdate"
	"gopkg.in/toast.v1"
)

var AWSALB string
var AWSALBCORS string
var JSESSIONID string
var NSC_wt_ena_tijccpmfui_ttm string
var RelayState string
var SAMLResponse string
var shibsessionName string
var shibsessionValue string
var JSESSIONID2 string

var newsList []map[string]string
var notiList []map[string]string
var (
	log      []byte
	home_log []byte
)

var funcList []func(client *http.Client, u *url.URL)

func ReadNotification(client *http.Client, u *url.URL) {
	req, _ := http.NewRequest("GET", "https://hoppii.hosei.ac.jp:443/portal/tool/474a4523-3b7b-42ac-a461-d8753982d3b6?panel=Main", nil)
	cookies := []*http.Cookie{
		{Name: "AWSALB", Value: AWSALB},
		{Name: "AWSALBCORS", Value: AWSALBCORS},
		{Name: shibsessionName, Value: shibsessionValue},
		{Name: "JSESSIONID", Value: JSESSIONID2},
	}
	client.Jar.SetCookies(u, cookies)

	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	r := bytes.NewReader(body)
	doc, _ := goquery.NewDocumentFromReader(r)
	doc.Find("tr").Each(func(_ int, s *goquery.Selection) {
		sec := s.Find("th[headers='subject'] span[class='skip']")
		if len(sec.Nodes) != 0 {
			url, _ := s.Find("a").Attr("href")
			newsMap := map[string]string{
				"channel": strings.TrimSpace(s.Find("td[headers='channel']").Text()),
				"subject": sec.Text(),
				"url":     strings.Replace(url, "&", "&amp;", -1),
			}
			newsList = append(newsList, newsMap)
		}
	})

	for i, d := range newsList {
		notification := toast.Notification{
			AppID:   "Hoppii通知君",
			Title:   d["channel"],
			Message: d["subject"],
			Actions: []toast.Action{
				{Type: "protocol", Label: "ブラウザで開く", Arguments: d["url"]},
			},
		}

		sha1 := sha1.Sum([]byte(d["channel"] + d["subject"]))

		if reflect.DeepEqual(log, sha1[:]) {
			if i == 0 {
				// notification := toast.Notification{
				// 	AppID:   "Hoppii通知君",
				// 	Title:   "なし",
				// 	Message: "最新の状態です",
				// }
				// notification.Push()
				fmt.Println("最新の状態です")
			}
			break
		}
		if i == 0 {
			file, err := os.Create("log.log")
			if err != nil {
				panic(err)
			}
			defer file.Close()
			file.Write(sha1[:])
		}
		notification.Push()
	}
}

func GetHomeWork(client *http.Client, u *url.URL) {
	homeHash := make(map[string]string)
	if len(home_log) != 0 {
		for _, row := range strings.Split(string(home_log), "\n") {
			p := strings.Split(string(row), ",")
			if len(p) > 1 {
				homeHash[p[0]] = p[1]
			}
		}
	}
	req, _ := http.NewRequest("GET", "https://hoppii.hosei.ac.jp:443/portal/tool/e3158cab-276f-48ca-97f2-83b3196afb4d?panel=Main", nil)
	cookies := []*http.Cookie{
		{Name: "AWSALB", Value: AWSALB},
		{Name: "AWSALBCORS", Value: AWSALBCORS},
		{Name: shibsessionName, Value: shibsessionValue},
		{Name: "JSESSIONID", Value: JSESSIONID2},
	}
	client.Jar.SetCookies(u, cookies)

	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	r := bytes.NewReader(body)
	doc, _ := goquery.NewDocumentFromReader(r)
	doc.Find("tr").Each(func(_ int, s *goquery.Selection) {
		if len(s.Find("td").Nodes) != 0 {
			var siteName string
			var title string
			var url string
			var dueDate string

			s.Find("td").Each(func(_ int, ss *goquery.Selection) {
				name, _ := ss.Attr("headers")
				if name == "siteName" {
					siteName = ss.Text()
				}
				if name == "title" {
					title = ss.Text()
					url, _ = s.Find("a").Attr("href")
				}
				if name == "dueDate" {
					dueDate = ss.Text()
				}

			})
			newsMap := map[string]string{
				"channel": strings.TrimSpace(siteName),
				"subject": strings.TrimSpace(title),
				"due":     strings.TrimSpace(dueDate),
				"url":     strings.Replace(url, "&", "&amp;", -1),
			}
			notiList = append(notiList, newsMap)
		}
	})
	for i, d := range notiList {
		notification := toast.Notification{
			AppID:   "Hoppii通知君",
			Title:   d["channel"],
			Message: d["subject"] + "\n----" + d["due"],
			Actions: []toast.Action{
				{Type: "protocol", Label: "ブラウザで開く", Arguments: d["url"]},
			},
		}

		sha1 := sha1.Sum([]byte(d["channel"] + d["subject"]))
		sha1String := hex.EncodeToString(sha1[:])
		if _, isE := homeHash[sha1String]; isE {
			homeHash[sha1String] = d["due"]
			due, _ := httpdate.Str2Time(d["due"], nil)
			leftTime := due.Sub(time.Now()).Hours()
			if leftTime < 24 && leftTime > 0 {
				notification.Push()
			}
		} else {
			homeHash[sha1String] = d["due"]
			notification.Push()
		}

		if reflect.DeepEqual(log, sha1[:]) {
			if i == 0 {
				// notification := toast.Notification{
				// 	AppID:   "Hoppii通知君",
				// 	Title:   "なし",
				// 	Message: "最新の状態です",
				// }
				// notification.Push()
				fmt.Println("最新の状態です")
			}
			break
		}
		// notification.Push()
	}
	file, err := os.Create("home_log.log")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	for key, val := range homeHash {
		file.WriteString(key + "," + val + "\n")
	}

}

func MessageErr(c string) {
	notification := toast.Notification{
		AppID:   "Hoppii通知君",
		Title:   "エラー",
		Message: c,
	}
	notification.Push()
}
func main() {
	client := &http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	flag.Parse()
	if len(flag.Args()) > 1 {
		MessageErr("引数が多すぎです")
	}

	if len(flag.Args()) == 0 {
		funcList = append(funcList, GetHomeWork)
		funcList = append(funcList, ReadNotification)
	}
	for _, arg := range flag.Args() {
		switch arg {
		case "kadai":
			funcList = append(funcList, GetHomeWork)
		case "home":
			funcList = append(funcList, ReadNotification)
		default:
			funcList = append(funcList, ReadNotification)
		}
	}

	st, err := ioutil.ReadFile("login.txt")
	if err != nil {
		MessageErr("login.txt読み込みエラー")
		panic(err)
	}
	StudentNumber := strings.Split(string(st), ",")[0]
	Password := strings.Split(string(st), ",")[1]

	log, err = ioutil.ReadFile("log.log")
	if err != nil {
		fmt.Println("log.logを新規作成")
		file, _ := os.Create("log.log")
		file.Close()
	}
	home_log, err = ioutil.ReadFile("home_log.log")
	if err != nil {
		fmt.Println("home_log.logを新規作成")
		file, _ := os.Create("home_log.log")
		file.Close()
	}

	//スタート、AWSALBとAWSALBCORSが手に入る
	req, _ := http.NewRequest("GET", "https://hoppii.hosei.ac.jp/sakai-login-tool/container", nil)
	resp, _ := client.Do(req)

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "AWSALB" {
			AWSALB = cookie.Value
		}
		if cookie.Name == "AWSALBCORS" {
			AWSALBCORS = cookie.Value
		}
	}

	//リダイレクト1回目、JSESSIONIDとNSC_wt_ena_tijccpmfui_ttmが手に入る
	jar, _ := cookiejar.New(nil)
	client = &http.Client{Jar: jar}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	NextUrl := resp.Header["Location"][0]
	//リダイレクト2回目JSESSIONIDとNSC_wt_ena_tijccpmfui_ttmが必要、ユーザー名とパスワードを入力するページが手に入る
	req, _ = http.NewRequest("GET", NextUrl, nil)
	resp, _ = client.Do(req)
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "JSESSIONID" {
			JSESSIONID = cookie.Value
		}
		if cookie.Name == "NSC_wt_ena_tijccpmfui_ttm" {
			NSC_wt_ena_tijccpmfui_ttm = cookie.Value
		}
	}
	u, _ := url.Parse(NextUrl)
	NextUrl = "https://" + u.Host + resp.Header["Location"][0]
	req, _ = http.NewRequest("GET", NextUrl, nil)
	client.Do(req)

	data := url.Values{"j_username": {StudentNumber}, "j_password": {Password}, "_eventId_proceed": {""}}

	//データを入力してPOSTする、JSESSIONIDとNSC_wt_ena_tijccpmfui_ttmが必要=>SSOのハッシュデータが手に入る
	req, _ = http.NewRequest("POST", NextUrl,
		strings.NewReader(data.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, _ = client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	//得られたHTMLをパースしてハッシュを手に入れる
	r := bytes.NewReader(body)
	doc, _ := goquery.NewDocumentFromReader(r)
	doc.Find("input[type='hidden']").Each(func(_ int, s *goquery.Selection) {
		band, ok := s.Attr("name")
		if ok {
			value, _ := s.Attr("value")
			if band == "RelayState" {
				RelayState = value
			}
			if band == "SAMLResponse" {
				SAMLResponse = value
			}
		}
	})
	//エラーチェック
	if RelayState == "" || SAMLResponse == "" {
		MessageErr("パスワードが違います")
		fmt.Println(resp.Header)
		fmt.Println(body)
		os.Exit(1)
	}

	//ハッシュデータをそのままPOSTする、shibsessionが得られる
	data = url.Values{"SAMLResponse": {SAMLResponse}, "RelayState": {RelayState}}
	req, _ = http.NewRequest("POST", "https://hoppii.hosei.ac.jp/Shibboleth.sso/SAML2/POST",
		strings.NewReader(data.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	u, _ = url.Parse("https://hoppii.hosei.ac.jp/Shibboleth.sso/SAML2/POST")
	cookies := []*http.Cookie{
		{Name: "AWSALB", Value: AWSALB},
		{Name: "AWSALBCORS", Value: AWSALBCORS},
	}
	client.Jar.SetCookies(u, cookies)
	resp, _ = client.Do(req)

	NextUrl = resp.Header["Location"][0]

	for _, cookie := range resp.Cookies() {
		if strings.Contains(cookie.Name, "shibsession") {
			shibsessionName = cookie.Name
			shibsessionValue = cookie.Value
		}
	}

	//リダイレクト1、JSESSIONID2が得られる
	req, _ = http.NewRequest("GET", NextUrl, nil)
	resp, _ = client.Do(req)
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "JSESSIONID" {
			JSESSIONID2 = cookie.Value
		}
	}

	//メインコンテンツにアクセス。AWSALB、AWSALBCORS、shibsessionName、JSESSIONIDが必要
	for _, f := range funcList {
		f(client, u)
	}

}
