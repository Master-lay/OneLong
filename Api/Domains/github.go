package Domains

import (
	"OneLong/Utils"
	outputfile "OneLong/Utils/OutPutfile"
	"OneLong/Utils/gologger"
	"crypto/tls"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"net/http"
	"regexp"
	"strings"
	//"strconv"
	//"strings"
	"time"
)

// 用于保护 addedURLs
func GetEnInfoGithub(response string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {
	respons := gjson.Get(response, "passive_dns").Array()

	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "Github"
	ensOutMap := make(map[string]*outputfile.ENSMap)
	for k, v := range GetENMap() {
		ensOutMap[k] = &outputfile.ENSMap{Name: v.Name, Field: v.Field, KeyWord: v.KeyWord}
	}

	addedURLs := make(map[string]bool)
	for aa, _ := range respons {
		ResponseJia := respons[aa].String()
		url := gjson.Parse(ResponseJia).Get("hostname").String()
		DomainsIP.Domains = append(DomainsIP.Domains, url)
		// 检查是否已存在相同的 URL
		if !addedURLs[url] {
			// 如果不存在重复则将 URL 添加到 Infos["Urls"] 中，并在 map 中标记为已添加
			ensInfos.Infos["Urls"] = append(ensInfos.Infos["Urls"], gjson.Parse(ResponseJia))
			addedURLs[url] = true
		}

	}

	//命令输出展示

	var data [][]string
	var keyword []string
	for _, y := range GetENMap() {
		for _, ss := range y.KeyWord {
			if ss == "数据关联" {
				continue
			}
			keyword = append(keyword, ss)
		}

		for _, res := range ensInfos.Infos["Urls"] {
			results := gjson.GetMany(res.Raw, y.Field...)
			var str []string
			for _, s := range results {
				str = append(str, s.String())
			}
			data = append(data, str)
		}

	}

	Utils.DomainTableShow(keyword, data, "Github")

	//zuo := strings.ReplaceAll(response, "[", "")
	//you := strings.ReplaceAll(zuo, "]", "")

	//ensInfos.Infos["hostname"] = append(ensInfos.Infos["hostname"], gjson.Parse(Result[1].String()))
	//getCompanyInfoById(pid, 1, true, "", options.Getfield, ensInfos, options)
	return ensInfos, ensOutMap

}

func Github(domain string, options *Utils.LongOptions, DomainsIP *outputfile.DomainsIP) string {

	var Hostname []string
	for add := 1; add < 11; add += 1 {
		urls := fmt.Sprintf("https://api.github.com/search/code?q=%s&per_page=100&page=%d&sort=indexed&&order=asc", domain, add)

		client := resty.New()
		client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
		client.SetTimeout(time.Duration(options.TimeOut) * time.Minute)
		if options.Proxy != "" {
			client.SetProxy(options.Proxy)
		}
		Authorization := " token " + options.LongConfig.Cookies.Github
		client.Header = http.Header{
			"User-Agent":    {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"},
			"Accept":        {"application/vnd.github.v3.text-match+json"},
			"Authorization": {Authorization},
		}

		client.Header.Del("Cookie")

		//强制延时1s
		time.Sleep(3 * time.Second)
		//加入随机延迟
		time.Sleep(time.Duration(options.GetDelayRTime()) * time.Second)
		clientR := client.R()

		clientR.URL = urls
		resp, err := clientR.Get(urls)

		for attempt := 0; attempt < 4; attempt++ {
			if resp.RawResponse == nil || strings.Contains(string(resp.Body()), "API rate limit exceeded for") {
				resp, _ = clientR.Get(urls)
				time.Sleep(20 * time.Second)
			} else if resp.Body() != nil {
				break
			}
		}
		if resp.RawResponse == nil || err != nil && add == 1 {
			gologger.Errorf("Github 链接无法访问尝试切换代理 \n")
			return ""
		} else if err != nil && add != 1 {
			continue
		}
		if gjson.Get(string(resp.Body()), "total_count").Int() == 0 && add == 1 {
			//gologger.Labelf("github 未发现域名 %s\n", domain)
			return ""
		} else if len(gjson.Get(string(resp.Body()), "items").Array()) == 0 {
			break
		} else if gjson.Get(string(resp.Body()), "total_count").Int() == 0 && add != 1 {
			break
		}
		hostname := `(?:[a-z0-9](?:[a-z0-9\-]{0,61}[a-z0-9])?\.)+` + regexp.QuoteMeta(domain)
		re := regexp.MustCompile(hostname)
		matches := re.FindAllStringSubmatch(string(resp.Body()), -1)
		for _, aa := range matches {
			if strings.Contains(aa[0], domain) {
				Hostname = append(Hostname, aa[0])
			}
		}
	}
	Hostname = Utils.SetStr(Hostname)
	result := "{\"passive_dns\":["
	var add int
	for add = 0; add < len(Hostname); add++ {
		result += "{\"hostname\"" + ":" + "\"" + Hostname[add] + "\"" + "},"
		DomainsIP.Domains = append(DomainsIP.Domains, Hostname[add])
	}
	result = result + "]}"
	res, ensOutMap := GetEnInfoGithub(result, DomainsIP)

	outputfile.MergeOutPut(res, ensOutMap, "Github", options)

	return "Success"
}
