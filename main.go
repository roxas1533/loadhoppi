package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"reflect"
	"strings"

	"github.com/PuerkitoBio/goquery"
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

func main() {
	client := &http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	st, err := ioutil.ReadFile("login.txt")
	if err != nil {
		panic(err)
	}
	StudentNumber := strings.Split(string(st), ",")[0]
	Password := strings.Split(string(st), ",")[1]

	log, err := ioutil.ReadFile("log.log")
	if err != nil {
		panic(err)
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
	req, _ = http.NewRequest("GET", "https://hoppii.hosei.ac.jp:443/portal/tool/474a4523-3b7b-42ac-a461-d8753982d3b6?panel=Main", nil)
	cookies = []*http.Cookie{
		{Name: "AWSALB", Value: AWSALB},
		{Name: "AWSALBCORS", Value: AWSALBCORS},
		{Name: shibsessionName, Value: shibsessionValue},
		{Name: "JSESSIONID", Value: JSESSIONID2},
	}
	client.Jar.SetCookies(u, cookies)

	resp, _ = client.Do(req)
	body, _ = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	r = bytes.NewReader(body)
	doc, _ = goquery.NewDocumentFromReader(r)
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
		// err = notification.Push()
		_ = notification
		if err != nil {
			panic(err)
		}
		if i == 0 {
			sha1 := sha1.Sum([]byte(d["channel"] + d["subject"]))
			fmt.Println(reflect.DeepEqual(log, sha1[:]))
		}
	}

}
