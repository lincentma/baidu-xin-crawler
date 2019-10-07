package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"

	"fmt"
	//"github.com/360EntSecGroup-Skylar/excelize"
	"strconv"
	//"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/gocolly/colly"
	"github.com/robertkrimen/otto"
	//"github.com/gocolly/colly/queue"
	"io"
	"log"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"
)

// 公司详情数据
type Company struct {
	// 搜索公司名称
	SearchName string
	// 公司姓名
	Name string
	// 公司地址
	Address string
	// 公司简介
	Desc string
	// 公司年报时间
	Annual string
	// 公司PID
	Pid string
	// 公司邮箱
	Email string
	// 公司类型
	EntType string
	// 所属行业
	Industry string
	// 统一社会信用代码
	UnifiedCode string
	// 公司URL
	Url string
}

// 公司数据API数据解析
type CompanyAPIResponse struct {
	Status int                    `json:"status"`
	Msg    string                 `json:"message"`
	Data   map[string]interface{} `json:"data"`
}

func main() {
	// 基本思路 串行获取
	errMsg := ""
	// 1. 读取公司名称数据
	companyNameFilePath := "./baidu_credit_company_name.conf"
	companyNameList, err := readAllFromFile(companyNameFilePath)
	if err != nil {
		errMsg = "获取公司名称列表失败"
		log.Println(errMsg)
		return
	}
	// 公司查询数据
	//companyNameList = companyNameList[:1]
	// 公司结果数据
	// 设置为数组
	// key => [{},{},{}]
	// key对应数组为空，空写入一行
	// 按照Excel格式添加写对应各列
	// companyDataList := ""

	// 创建csv文件
	companyDetailFilePath := "./baidu_credit_company_result.csv"
	_, err = createCompanyCsvFile(companyDetailFilePath)
	if err != nil {
		errMsg = "【4】创建公司查询结果CSV文件"
		errMsg += "——"
		errMsg += err.Error()
		log.Println(errMsg)
		fmt.Println(errMsg)
		return
	}

	// 2. 遍历所有公司名称
	for index, companyName := range companyNameList {
		// 2.1 获取查询公司名称的URL
		companyNameSearchUrl, err := getCompanyNameSearchUrl(companyName)
		if err != nil {
			errMsg = "【1】获取公司查询URL失败"
			errMsg += "——"
			errMsg += companyName
			log.Println(errMsg)
			return
		}

		// 2.2 请求搜索页面，并解析查询结果
		fmt.Println(companyNameSearchUrl)
		companyDetailUrlList, err := getCompanyNameSearchPageData(companyNameSearchUrl)
		if err != nil {
			errMsg = "【2】获取公司详情URL失败"
			errMsg += "——"
			errMsg += companyName
			log.Println(errMsg)
			return
		}

		// 遍历获取所有搜索结果
		for _, companyDetailUrl := range companyDetailUrlList {
			// 请求解析公司详情数据
			// 思路：1. 获取公司详情Html中的数据 2. 获取Html中的baiducode , tk参数进行计算tot参数
			companyPid, companyBaiduCode, companyTk, err := getCompanyDetailSearchPageData(companyDetailUrl, companyNameSearchUrl)
			fmt.Println(companyBaiduCode)
			fmt.Println(companyTk)
			if err != nil {
				errMsg = "【2】获取公司详情页面失败"
				errMsg += "——"
				errMsg += companyDetailUrl
				log.Println(errMsg)
				return
			}
			// 运行js函数计算tot参数
			companyTot, err := getCompanyAPIRequestParam(companyTk, companyBaiduCode)
			if err != nil {
				errMsg = "【3】获取公司详情API请求参数失败"
				errMsg += "——"
				errMsg += err.Error()
				log.Println(errMsg)
				return
			}

			// 获取公司基本信息
			// 公司名称
			companyEntName := "暂无"
			// 公司地址
			companyAddress := "暂无"
			// 公司简介
			companyDesc := "暂无"
			// 公司邮箱
			companyEmail := "暂无"
			// 所属行业
			companyIndustry := "暂无"
			// 统一社会信用代码
			companyUnifiedCode := "暂无"
			// 企业类型
			companyEntType := "暂无"
			// 公司年报时间
			companyAnnualYear := "暂无"

			companyBasicApiUri := "https://xin.baidu.com/detail/basicAjax?"
			companyBasicApiUrl, _ := getCompanyApiUrl(companyBasicApiUri, companyPid, companyTot)
			// 请求获取数据
			companyBasicDataResponse, _ := getCompanyRequestData(companyBasicApiUrl, companyDetailUrl)
			companyEntName, companyAddress, companyDesc, companyEmail, companyUnifiedCode, companyEntType, companyIndustry, _ = getCompanyBasicData(companyBasicDataResponse)

			// 获取公司年报信息
			companyAnnualApiUri := "https://xin.baidu.com/detail/annualListAjax?"
			companyAnnualApiUrl, _ := getCompanyApiUrl(companyAnnualApiUri, companyPid, companyTot)
			// 请求获取数据
			companyAnnualDataResponse, _ := getCompanyRequestData(companyAnnualApiUrl, companyDetailUrl)
			companyAnnualYear, _ = getCompanyAnnualData(companyAnnualDataResponse)

			companyDetailData := Company{}
			// 构建空数据
			companyDetailData.SearchName = companyName
			companyDetailData.Name = companyEntName
			companyDetailData.Address = companyAddress
			companyDetailData.Desc = companyDesc
			companyDetailData.Annual = companyAnnualYear
			companyDetailData.Pid = companyPid
			companyDetailData.Email = companyEmail
			companyDetailData.EntType = companyEntType
			companyDetailData.Industry = companyIndustry
			companyDetailData.UnifiedCode = companyUnifiedCode
			companyDetailData.Url = companyDetailUrl

			// 2.7 整理数据；写入文件
			// 写入CSV
			_,err = writeCompanyCsvFile(companyDetailFilePath,
				companyDetailData.SearchName,
				companyDetailData.Name,
				companyDetailData.Address,
				companyDetailData.Desc,
				companyDetailData.Email,
				companyDetailData.UnifiedCode,
				companyDetailData.EntType,
				companyDetailData.Industry,
				companyDetailData.Annual,
				companyDetailData.Url)

			// 暂停3s
			time.Sleep(1000000000)
		}
		// 判断页面查询是否有结果
		if len(companyDetailUrlList) == 0 {
			companyDetailData := Company{}
			// 构建空数据
			companyDetailData.SearchName = companyName
			companyDetailData.Name = "暂无"
			companyDetailData.Address = "暂无"
			companyDetailData.Desc = "暂无"
			companyDetailData.Annual = "暂无"
			companyDetailData.Pid = "暂无"
			companyDetailData.Email = "暂无"
			companyDetailData.EntType = "暂无"
			companyDetailData.Industry = "暂无"
			companyDetailData.UnifiedCode = "暂无"
			companyDetailData.Url = "暂无"

			// 写入CSV
			_,err = writeCompanyCsvFile(companyDetailFilePath,
				companyDetailData.SearchName,
				companyDetailData.Name,
				companyDetailData.Address,
				companyDetailData.Desc,
				companyDetailData.Email,
				companyDetailData.UnifiedCode,
				companyDetailData.EntType,
				companyDetailData.Industry,
				companyDetailData.Annual,
				companyDetailData.Url)
		}

		// 暂停时间避免快速访问
		// 暂停3s
		if index >= 30 && index % 30 == 0 {
			fmt.Println("休息一下")
			time.Sleep(1000000000 * 60)
		} else {
			time.Sleep(1000000000)
		}

	}
}

// 文件读取
func readAllFromFile(fileName string) ([]string, error) {
	f, err := os.Open(fileName)
	var nameList []string
	if err != nil {
		log.Println("Open File Error:", err)
		return nil, err
	}
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			nameList = append(nameList, line)
			//g.Tasks <- line
		}
		if err != nil {
			if err == io.EOF {
				log.Println("Read File Finish")
				//close(g.Tasks)
				return nameList, nil
			}
			log.Println("Read File Error:", err)
			return nil, err
		}
	}
	return nil, err
}


// 判断文件是否存在
func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

// 获取公司名称搜索URL
func getCompanyNameSearchUrl(companyName string) (string, error) {
	companyNameSearchUrl := "https://xin.baidu.com/s"

	// 防止跳转输入验证码的措施
	queryArr := make(map[string]string)
	// 1. q=公司名称
	queryArr["q"] = companyName
	// 2. v=当前时间（毫秒）戳
	currentMicroTime := time.Now().UnixNano() / 1e6
	queryArr["v"] = strconv.FormatInt(currentMicroTime, 10)
	// 3. t=0
	queryArr["t"] = "0"
	// 4. castk=LTE=
	queryArr["castk"] = "LTE="
	// 5. fl=1
	queryArr["fl"] = "1"

	// 打乱顺序，并没有什么用
	queryKeyArr := []string{"q", "v", "t", "castk", "fl"}
	queryKeyArr = arrKeyShuffle(queryKeyArr)
	// URL拼接
	queryUrl := url.Values{}
	// 遍历获取
	for _, queryKey := range queryKeyArr {
		queryUrl.Add(queryKey, queryArr[queryKey])
	}
	// URL编码，自动按照参数首字母排序
	queryUrlStr := queryUrl.Encode()

	companyNameSearchUrl = companyNameSearchUrl + "?" + queryUrlStr
	// 调试记录
	logString := "【公司名称查询地址】：" + companyName + " " + queryUrlStr
	fmt.Println(logString)

	return companyNameSearchUrl, nil

}

// 随机排序
func arrKeyShuffle(currentArray []string) ([]string) {

	len := len(currentArray)
	randomArray := make([]string, len)
	for i := 0; i < len; i++ {
		randomArray[i] = currentArray[i]
	}
	var (
		pos  int
		temp string
	)
	for i := len - 1; i >= 0; i-- {
		pos = rand.Intn(i + 1)
		temp = randomArray[pos]
		randomArray[pos] = randomArray[i]
		randomArray[i] = temp
	}
	return randomArray
}

// 获取公司搜索页面结果
func getCompanyNameSearchPageData(companyNameSearchUrl string) ([]string, error) {

	var companyDetailUrlList []string

	// Instantiate default collector
	c := colly.NewCollector(
		// 搜索页面域名
		colly.AllowedDomains("xin.baidu.com"),
		// 异步请求
		//colly.Async(true),
		// 模拟浏览器
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36"),
		// 缓存处理
		colly.CacheDir("./company_name_search_cache"),
		// cookie
	)

	// 请求频率限制
	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*xin.*",
		Parallelism: 1,
		// 间隔3s请求一次
		Delay: 5 * time.Second,
	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	// 请求前
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("【Colly请求URL】", r.URL.String())
		// 根据真实网页参数进行相同配置
		r.Headers.Set("Host", "xin.baidu.com")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")
		r.Headers.Set("Referer", "https://xin.baidu.com/")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Accept-Language", "zh-CN,zh;q=0.9")
		// cookie
		r.Headers.Set("Cookie", "BAIDUID=1F0901EA476FE22B175C04161C1B0F84:FG=1; BIDUPSID=1F0901EA476FE22B175C04161C1B0F84; PSTM=1561879879; log_guid=fcec05814b6866370138bc5f142051a4; __cas__st__=NLI; __cas__id__=0; H_PS_PSSID=1465_21091_29522_29519_29099_29568_29220_29071; BDORZ=B490B5EBF6F3CD402E515D22BCDA1598; delPer=0; PSINO=5; Hm_lvt_baca6fe3dceaf818f5f835b0ae97e4cc=1564605760,1566632615,1566632702; ZX_UNIQ_UID=06d7832f8446f9b8d87ae6b21e362425; ZX_HISTORY=%5B%7B%22visittime%22%3A%222019-08-28+08%3A33%3A20%22%2C%22pid%22%3A%22xlTM-TogKuTwTIaX%2A6m6UlGKgOyurTbS4Qmd%22%7D%2C%7B%22visittime%22%3A%222019-08-28+08%3A06%3A08%22%2C%22pid%22%3A%22xlTM-TogKuTwVm37thtYv809P%2AZ8Xj4jlAmd%22%7D%2C%7B%22visittime%22%3A%222019-08-28+01%3A36%3A13%22%2C%22pid%22%3A%22xlTM-TogKuTw0U2RdoELgmZirD8IPyQ5jwmd%22%7D%2C%7B%22visittime%22%3A%222019-08-28+01%3A13%3A42%22%2C%22pid%22%3A%22xlTM-TogKuTwIfkSiSjfPOzJIw210adcqQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-26+02%3A26%3A20%22%2C%22pid%22%3A%22xlTM-TogKuTwiU1mkxvYExr9cxLBkOvwBQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-25+14%3A14%3A57%22%2C%22pid%22%3A%22xlTM-TogKuTw3eCYt2S11GJ482Vurf4UhQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-24+15%3A44%3A23%22%2C%22pid%22%3A%22xlTM-TogKuTwuqQimcOUeIY%2AbwBWCvNDtAmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A13%3A21%22%2C%22pid%22%3A%22xlTM-TogKuTwQ07YFJdxlMVEXq0J7P4Vzgmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A13%3A09%22%2C%22pid%22%3A%22xlTM-TogKuTwdu3igAP3ZsXj-tFNCsKIQQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A12%3A41%22%2C%22pid%22%3A%22xlTM-TogKuTwHaFr6dOjT47sI91-qNYCngmd%22%7D%5D; Hm_lpvt_baca6fe3dceaf818f5f835b0ae97e4cc=1566952617")
	})

	// 页面处理
	c.OnHTML(".zx-result-counter", func(e *colly.HTMLElement) {
		searchCount := e.Text
		// 判断查询结果
		fmt.Printf("【公司查询页面】查询结果数量: %q", searchCount)
		fmt.Println()
	})
	c.OnHTML(".zx-list-item-url", func(e *colly.HTMLElement) {
		companyDetailUri := e.Attr("href")
		// 转换为公司详情链接
		companyDetailUrl, err := getCompanyDetailSearchUrl(companyDetailUri)
		if err != nil {
			log.Println("【公司查询页面】解析公司详情链接失败")
		}
		companyDetailUrlList = append(companyDetailUrlList, companyDetailUrl);
	})

	// 响应处理
	c.OnResponse(func(r *colly.Response) {
		// 保存查询页面Html文件
		// 提取文档搜索名称
		companyName := ""
		urlData, err := url.Parse(r.Request.URL.String())
		if err != nil {
			log.Println("【公司查询页面】解析公司名称失败")
		} else {
			queryMap, _ := url.ParseQuery(urlData.RawQuery)
			if len(queryMap["q"]) > 0 {
				companyName = queryMap["q"][0]
			} else {
				companyName = "暂无" + string(time.Now().Unix())
			}
		}
		htmlFileName := "./company_name_search_html/" + companyName + ".html"
		isHtmlFileExist := checkFileIsExist(htmlFileName)
		fmt.Println(htmlFileName)
		if false == isHtmlFileExist {
			os.Create(htmlFileName)
		}
		r.Save(htmlFileName)
	})

	//结束
	c.OnScraped(func(r *colly.Response) {
		fmt.Println("【请求结束】", r.Request.URL)
	})

	// 添加访问地址
	c.Visit(companyNameSearchUrl)

	return companyDetailUrlList, nil
}

func getCompanyDetailSearchUrl(searchDetailUrl string) (string, error) {
	companyDetailSearchUrl := "https://xin.baidu.com"
	companyDetailSearchUrl += searchDetailUrl

	// 防止跳转输入验证码的措施
	queryArr := make(map[string]string)
	// 1. v=当前时间（毫秒）戳
	currentMicroTime := time.Now().UnixNano() / 1e6
	queryArr["_"] = strconv.FormatInt(currentMicroTime, 10)
	// 3. t=0
	queryArr["t"] = "0"
	// 4. castk=LTE=
	queryArr["castk"] = "LTE="
	// 5. fl=1
	queryArr["fl"] = "1"

	// 打乱顺序，并没有什么用
	queryKeyArr := []string{"_", "t", "castk", "fl"}
	queryKeyArr = arrKeyShuffle(queryKeyArr)
	// URL拼接
	queryUrl := url.Values{}
	// 遍历获取
	for _, queryKey := range queryKeyArr {
		queryUrl.Add(queryKey, queryArr[queryKey])
	}
	// URL编码，自动按照参数首字母排序
	queryUrlStr := queryUrl.Encode()
	// 原本URL中已经存在?
	companyDetailSearchUrl = companyDetailSearchUrl + "&" + queryUrlStr
	// 调试记录
	logString := "【公司详情查询地址】：" + companyDetailSearchUrl
	fmt.Println(logString)

	return companyDetailSearchUrl, nil
}

// 获取公司搜索页面结果
func getCompanyDetailSearchPageData(companyDetailSearchUrl string, companyNameSearchUrl string) (string, string, string, error) {

	var companyName = ""
	var companyBaiduCode = ""
	var companyTk = ""
	// Tk对应Html中的ID标签
	var companyTkId = ""
	// Tk对应Html的Attr属性
	var companyTkAtrr = ""
	// Pid
	var companyPid = ""

	// Instantiate default collector
	c := colly.NewCollector(
		// 搜索页面域名
		colly.AllowedDomains("xin.baidu.com"),
		// 异步请求
		//colly.Async(true),
		// 模拟浏览器
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36"),
		// 缓存处理
		colly.CacheDir("./company_detail_search_cache"),
		// cookie
	)

	// 请求频率限制
	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*xin.*",
		Parallelism: 1,
		// 间隔3s请求一次
		Delay: 5 * time.Second,
	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	// 请求前
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("【Colly请求URL】", r.URL.String())
		// 获取Pid
		urlData, err := url.Parse(r.URL.String())
		if err != nil {
			log.Println("【公司详情页面】解析公司详情URL失败")
		} else {
			queryMap, _ := url.ParseQuery(urlData.RawQuery)
			if len(queryMap["pid"]) > 0 {
				companyPid = queryMap["pid"][0]
			} else {
				companyPid = "暂无" + string(time.Now().Unix())
			}
		}
		// 根据真实网页参数进行相同配置
		r.Headers.Set("Host", "xin.baidu.com")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")
		r.Headers.Set("Referer", companyNameSearchUrl)
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Accept-Language", "zh-CN,zh;q=0.9")
		// cookie
		//r.Headers.Set("Cookie", "BAIDUID=1F0901EA476FE22B175C04161C1B0F84:FG=1; BIDUPSID=1F0901EA476FE22B175C04161C1B0F84; PSTM=1561879879; log_guid=fcec05814b6866370138bc5f142051a4; __cas__st__=NLI; __cas__id__=0; H_PS_PSSID=1465_21091_29522_29519_29099_29568_29220_29071; BDORZ=B490B5EBF6F3CD402E515D22BCDA1598; delPer=0; PSINO=5; Hm_lvt_baca6fe3dceaf818f5f835b0ae97e4cc=1564605760,1566632615,1566632702; ZX_UNIQ_UID=06d7832f8446f9b8d87ae6b21e362425; ZX_HISTORY=%5B%7B%22visittime%22%3A%222019-08-28+01%3A36%3A03%22%2C%22pid%22%3A%22xlTM-TogKuTw0U2RdoELgmZirD8IPyQ5jwmd%22%7D%2C%7B%22visittime%22%3A%222019-08-28+01%3A13%3A42%22%2C%22pid%22%3A%22xlTM-TogKuTwIfkSiSjfPOzJIw210adcqQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-26+02%3A26%3A20%22%2C%22pid%22%3A%22xlTM-TogKuTwiU1mkxvYExr9cxLBkOvwBQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-25+14%3A14%3A57%22%2C%22pid%22%3A%22xlTM-TogKuTw3eCYt2S11GJ482Vurf4UhQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-24+15%3A44%3A23%22%2C%22pid%22%3A%22xlTM-TogKuTwuqQimcOUeIY%2AbwBWCvNDtAmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A13%3A21%22%2C%22pid%22%3A%22xlTM-TogKuTwQ07YFJdxlMVEXq0J7P4Vzgmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A13%3A09%22%2C%22pid%22%3A%22xlTM-TogKuTwdu3igAP3ZsXj-tFNCsKIQQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A12%3A41%22%2C%22pid%22%3A%22xlTM-TogKuTwHaFr6dOjT47sI91-qNYCngmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A12%3A26%22%2C%22pid%22%3A%22xlTM-TogKuTwvf52doZ-zgc7yQSY1i2D-gmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A12%3A07%22%2C%22pid%22%3A%22xlTM-TogKuTwxhvahEqjDV1IthwzskKqygmd%22%7D%5D; Hm_lpvt_baca6fe3dceaf818f5f835b0ae97e4cc=1566927365")
	})

	// 页面处理
	// 1- 公司名称
	c.OnHTML(".zx-detail-company", func(e *colly.HTMLElement) {
		companyName = e.ChildText(".zx-detail-company-title")
		// 去除"认领企业"
		companyName = strings.Replace(companyName, "认领企业", "", -1)
		fmt.Printf("查询公司名称: %q", companyName)
		fmt.Println()
	})
	// 2- BaiduCode
	c.OnHTML(".zx-detail-company-info", func(e *colly.HTMLElement) {
		companyBaiduCode = e.ChildText("#baiducode")
		// 去除"认领企业"
		fmt.Printf("查询公司BaiduCode: %q", companyBaiduCode)
		fmt.Println()
	})
	// 3- Tk对应的ID和Attr
	c.OnHTML("body", func(e *colly.HTMLElement) {
		// 遍历正则查询
		// 双斜杠保证转义字符
		tkParamsReg := regexp.MustCompile("var tk = document.getElementById\\(\\'(.*?)\\'\\).getAttribute\\(\\'(.*?)\\'\\);")

		// 获取脚本内容
		tkParams := tkParamsReg.FindStringSubmatch(e.Text)
		// 判断匹配是否为空
		if len(tkParams) > 2 {
			if tkParams[1] != "" {
				companyTkId = tkParams[1]
				fmt.Printf("查询公司TkID %q", companyTkId)
				fmt.Println()
			}
			if tkParams[2] != "" {
				companyTkAtrr = tkParams[2]
				fmt.Printf("查询公司TkAttr: %q", companyTkAtrr)
				fmt.Println()
			}

			// 正则获取获取TK值
			tkValuesReg := regexp.MustCompile(companyTkAtrr + "=" + "\"" + "(.*?)" + "\"")
			// 由于e.Text没有对应的Tk标签内容，所以搜索范围扩大到页面的所有内容
			tkValues := tkValuesReg.FindStringSubmatch(string(e.Response.Body))
			if len(tkValues) > 1 {
				if tkValues[1] != "" {
					companyTk = tkValues[1]
					fmt.Printf("查询公司Tk %q", companyTk)
					fmt.Println()
				}
			}
			// 通过获取对应的tk值, 但是标签每一次都有变化，存在获取不到值得情况
			//fmt.Println(e.DOM.Find("#" + companyTkId).Attr(companyTkAtrr))
		}
	})

	// 响应处理
	c.OnResponse(func(r *colly.Response) {
		// 由于onHtml处理会比OnResponse晚，无法获取更新过后的公司名称，所以保存Html文件放在结束

	})

	//结束
	c.OnScraped(func(r *colly.Response) {
		fmt.Println("【请求结束】", r.Request.URL)
		// 保存查询页面Html文件
		// 提取文档搜索名称
		htmlFileName := "./company_detail_search_html/" + companyName + ".html"
		isHtmlFileExist := checkFileIsExist(htmlFileName)
		fmt.Println(htmlFileName)
		if false == isHtmlFileExist {
			os.Create(htmlFileName)
		}
		r.Save(htmlFileName)

	})

	// 添加访问地址
	c.Visit(companyDetailSearchUrl)

	return companyPid, companyBaiduCode, companyTk, nil
}

// 获取公司信息API请求的tot参数
func getCompanyAPIRequestParam(companyTk string, companyBaiduCode string) (string, error) {
	var companyTot = ""
	vm := otto.New()
	// js计算函数
	jsScript := `function mix(tk, bid) {var tk = tk.split("").reverse().join("");return tk.substring(0, tk.length - bid.length);}`
	// 加载js函数
	vm.Run(jsScript)
	// 调用js函数
	jsReturnValue, err := vm.Call("mix", nil, companyTk, companyBaiduCode)
	// 获取返回值
	companyTot = jsReturnValue.String()

	return companyTot, err
}

// 获取
func getCompanyApiUrl(companyApiUri string, companyPid string, companyTot string) (string, error) {
	companyApiUrl := companyApiUri

	// 防止跳转输入验证码的措施
	queryArr := make(map[string]string)
	// 1. _=当前时间（毫秒）戳
	currentMicroTime := time.Now().UnixNano() / 1e6
	queryArr["_"] = strconv.FormatInt(currentMicroTime, 10)
	// 2. pid
	queryArr["pid"] = companyPid
	// 3. tot
	queryArr["tot"] = companyTot
	//// 4. castk=LTE=
	//queryArr["castk"] = "LTE="
	//// 5. fl=1
	//queryArr["fl"] = "1"

	queryUrl := url.Values{}
	// 遍历获取
	for queryKey, queryValue := range queryArr {
		queryUrl.Add(queryKey, queryValue)
	}
	// URL编码，自动按照参数首字母排序
	queryUrlStr := queryUrl.Encode()
	// 原本URL中已经存在?
	companyApiUrl = companyApiUrl + queryUrlStr
	// 调试记录
	logString := "【公司数据API地址】：" + companyApiUrl
	fmt.Println(logString)

	return companyApiUrl, nil
}

// API GET请求
func getCompanyRequestData(requestUrl string, refererUrl string) (CompanyAPIResponse, error) {

	client := &http.Client{}

	request, err := http.NewRequest("GET", requestUrl, nil)

	//增加header选项
	request.Header.Add("Host", "xin.baidu.com")
	request.Header.Add("Connection", "keep-alive")
	request.Header.Add("Accept", "application/json, text/javascript, */*; q=0.01")
	// 此项设置导致无法解析json
	//request.Header.Add("Accept-Encoding", "gzip, deflate, br")
	request.Header.Add("Accept-Language", "zh-CN,zh;q=0.9")
	request.Header.Add("Referer", refererUrl)
	request.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")
	request.Header.Add("Cookie", "BAIDUID=1F0901EA476FE22B175C04161C1B0F84:FG=1; BIDUPSID=1F0901EA476FE22B175C04161C1B0F84; PSTM=1561879879; log_guid=fcec05814b6866370138bc5f142051a4; __cas__st__=NLI; __cas__id__=0; H_PS_PSSID=1465_21091_29522_29519_29099_29568_29220_29071; BDORZ=B490B5EBF6F3CD402E515D22BCDA1598; delPer=0; PSINO=5; Hm_lvt_baca6fe3dceaf818f5f835b0ae97e4cc=1564605760,1566632615,1566632702; ZX_HISTORY=%5B%7B%22visittime%22%3A%222019-08-28+01%3A13%3A42%22%2C%22pid%22%3A%22xlTM-TogKuTwIfkSiSjfPOzJIw210adcqQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-26+02%3A26%3A20%22%2C%22pid%22%3A%22xlTM-TogKuTwiU1mkxvYExr9cxLBkOvwBQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-25+14%3A14%3A57%22%2C%22pid%22%3A%22xlTM-TogKuTw3eCYt2S11GJ482Vurf4UhQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-24+15%3A44%3A23%22%2C%22pid%22%3A%22xlTM-TogKuTwuqQimcOUeIY%2AbwBWCvNDtAmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A13%3A21%22%2C%22pid%22%3A%22xlTM-TogKuTwQ07YFJdxlMVEXq0J7P4Vzgmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A13%3A09%22%2C%22pid%22%3A%22xlTM-TogKuTwdu3igAP3ZsXj-tFNCsKIQQmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A12%3A41%22%2C%22pid%22%3A%22xlTM-TogKuTwHaFr6dOjT47sI91-qNYCngmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A12%3A26%22%2C%22pid%22%3A%22xlTM-TogKuTwvf52doZ-zgc7yQSY1i2D-gmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A12%3A07%22%2C%22pid%22%3A%22xlTM-TogKuTwxhvahEqjDV1IthwzskKqygmd%22%7D%2C%7B%22visittime%22%3A%222019-08-01+09%3A11%3A51%22%2C%22pid%22%3A%22xlTM-TogKuTwzLY6VDm4K6mLIdbF-doTWgmd%22%7D%5D; ZX_UNIQ_UID=06d7832f8446f9b8d87ae6b21e362425; Hm_lpvt_baca6fe3dceaf818f5f835b0ae97e4cc=1566927001")
	if err != nil {
		fmt.Println(err)
	}

	response, _ := client.Do(request)
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	fmt.Println("【API请求结果】", string(body))
	// 解析json
	resp := CompanyAPIResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		fmt.Println(err)
	}
	return resp, nil
}

// 公司基础数据
func getCompanyBasicData(companyAPIResponse CompanyAPIResponse) (string, string, string, string, string, string, string, error) {

	companyBasicData := companyAPIResponse.Data
	// struct 转为数组
	// 公司名称
	companyName := "暂无"
	// 公司地址
	companyAddress := "暂无"
	// 公司简介
	companyDesc := "暂无"
	// 公司邮箱
	companyEmail := "暂无"
	// 所属行业
	companyIndustry := "暂无"
	// 统一社会信用代码
	companyUnifiedCode := "暂无"
	// 企业类型
	companyEntType := "暂无"

	if companyBasicData["entName"] != nil {
		companyName = companyBasicData["entName"].(string)
		fmt.Println("【公司名称】", companyName)
	}
	if companyBasicData["regAddr"] != nil {
		companyAddress = companyBasicData["regAddr"].(string)
		fmt.Println("【公司地址】", companyAddress)
	}
	if companyBasicData["describe"] != nil {
		companyDesc = companyBasicData["describe"].(string)
		fmt.Println("【公司简介】", companyDesc)
	}
	if companyBasicData["email"] != nil {
		companyEmail = companyBasicData["email"].(string)
		fmt.Println("【公司邮箱】", companyEmail)
	}
	if companyBasicData["unifiedCode"] != nil {
		companyUnifiedCode = companyBasicData["unifiedCode"].(string)
		fmt.Println("【公司统一信用代码】", companyUnifiedCode)
	}
	if companyBasicData["entType"] != nil {
		companyEntType = companyBasicData["entType"].(string)
		fmt.Println("【公司类型】", companyEntType)
	}
	if companyBasicData["industry"] != nil {
		companyIndustry = companyBasicData["industry"].(string)
		fmt.Println("【公司所属行业】", companyIndustry)
	}

	return companyName, companyAddress, companyDesc, companyEmail, companyUnifiedCode, companyEntType, companyIndustry, nil
}

// 公司年报数据
func getCompanyAnnualData(companyAPIResponse CompanyAPIResponse) (string, error) {

	companyAnnualData := companyAPIResponse.Data
	// struct 转为数组
	// 判断年份是否为空
	newestAnnualYear := "暂无"
	if companyAnnualData["reportYears"] != nil {
		annualYears := companyAnnualData["reportYears"].([]interface{})
		// 最新一年数据,转换为string
		// 如果年份为空
		if len(annualYears) > 0 {
			newestAnnualYear = annualYears[0].(string)
		}
	}
	fmt.Println("【公司年报】", newestAnnualYear)

	return newestAnnualYear, nil
}

func createCompanyCsvFile(fileName string) (string, error) {
	f, err := os.Create(fileName)
	if err != nil {
		panic(err)
		fmt.Println(err.Error())
		return "", err
	}
	defer f.Close()
	// 写入UTF-8 BOM
	f.WriteString("\xEF\xBB\xBF")

	// 写入表头
	w := csv.NewWriter(f)
	w.Comma = '\t'
	w.UseCRLF = true
	w.Write([]string{"公司搜索名称", "公司名称", "公司地址", "公司简介", "联系邮箱","统一社会信用代码", "企业类型", "所属行业", "企业年报", "查询地址"})
	// 写文件需要flush，不然缓存满了，后面的就写不进去了，只会写一部分
	w.Flush()
	return "", nil
}


func writeCompanyCsvFile(fileName string,
	companySearchName string,
	companyName string,
	companyAddress string,
	companyDesc string,
	companyEmail string,
	companyUnifiedCode string,
	companyEntType string,
	companyIndustry string,
	companyAnnualYear string,
	companyUrl string) (string, error) {
	nfs, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("can not create file, err is %+v", err)
		fmt.Println(err.Error())
	}
	defer nfs.Close()
	nfs.Seek(0, io.SeekEnd)

	w := csv.NewWriter(nfs)

	w.Comma = '\t'
	w.UseCRLF = true
	row := []string{companySearchName, companyName, companyAddress, companyDesc, companyEmail, companyUnifiedCode, companyEntType, companyIndustry, companyAnnualYear, companyUrl}
	err = w.Write(row)
	if err != nil {
		log.Fatalf("can not create file, err is %+v", err)
		fmt.Println(err.Error())
	}
	w.Flush()

	return "", nil
}