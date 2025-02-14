package Domains

import (
	"OneLong/Utils"
	outputfile "OneLong/Utils/OutPutfile"
	"OneLong/Utils/gologger"
	"crypto/tls"
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
func GetEnInfoRapiddns(response string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {
	respons := gjson.Get(response, "passive_dns").Array()
	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "Rapiddns"
	ensOutMap := make(map[string]*outputfile.ENSMap)
	for k, v := range GetENMap() {
		ensOutMap[k] = &outputfile.ENSMap{Name: v.Name, Field: v.Field, KeyWord: v.KeyWord}
	}

	for aa, _ := range respons {
		ensInfos.Infos["Urls"] = append(ensInfos.Infos["Urls"], gjson.Parse(respons[aa].String()))
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

	Utils.DomainTableShow(keyword, data, "Rapiddns")

	return ensInfos, ensOutMap

}

func Rapiddns(domain string, options *Utils.LongOptions, DomainsIP *outputfile.DomainsIP) string {
	//gologger.Infof("Rapiddns Api搜索域名 \n")
	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	client.SetTimeout(time.Duration(options.TimeOut) * time.Minute)
	if options.Proxy != "" {
		client.SetProxy(options.Proxy)
		//client.SetProxy("192.168.203.111:1111")
	}
	urls := "https://rapiddns.io/subdomain/" + domain + "?full=1"
	client.Header = http.Header{
		"User-Agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"},
		"Accept":     {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"},
	}

	//强制延时1s
	time.Sleep(3 * time.Second)
	//加入随机延迟
	time.Sleep(time.Duration(options.GetDelayRTime()) * time.Second)
	clientR := client.R()
	clientR.URL = urls
	response, err := clientR.Get(urls)

	for add := 1; add < 4; add += 1 {
		if response.RawResponse == nil {
			response, _ = clientR.Get(urls)
			time.Sleep(3 * time.Second)
		} else if response.Body() != nil {
			break
		}
	}
	if err != nil {
		gologger.Errorf("Rapiddns API 链接访问失败尝试切换代理\n")
		return ""
	}
	//Total: <span style="color: #39cfca; ">0
	if strings.Contains(string(response.Body()), "Total: <span style=\"color: #39cfca; \">0") {
		//gologger.Labelf("Rapiddns Api未发现域名 %s\n", domain)
		return ""
	} else if strings.Contains(string(response.Body()), "404 NOT FOUND") {
		return ""
	}
	host := regexp.MustCompile(`<td>((?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,})</td>`)
	hostrea := host.FindAllStringSubmatch(strings.TrimSpace(string(response.Body())), -1)
	var Hostname []string
	// 使用 HTML 解析器解析 HTML 内容
	for _, bu := range hostrea {
		Hostname = append(Hostname, bu[1])
	}
	Hostname = Utils.SetStr(Hostname)
	// 查找具有特定 class 的元素并获取其内容
	//var Hostname []string

	var result string
	result = "{\"passive_dns\":["
	for i := 0; i < len(Hostname); i++ {
		result += "{\"hostname\"" + ":" + "\"" + Hostname[i] + "\"" + "},"
		DomainsIP.Domains = append(DomainsIP.Domains, Hostname[i])
	}
	result = result + "]}"
	res, ensOutMap := GetEnInfoRapiddns(result, DomainsIP)

	outputfile.MergeOutPut(res, ensOutMap, "Rapiddns", options)

	return "Success"
}
