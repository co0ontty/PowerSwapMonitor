package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/wanghuiyt/ding"
	"gopkg.in/ini.v1"
)

type taskInfo struct {
	DingtalkToken    string
	DingtalkSec      string
	PowerSwapInfoUrl string
	PowerSwapList    []PowerSwapIndex
	ChangeList       []PowerSwapIndex
}

type PowerMapInfo struct {
	RequestID   string `json:"request_id"`
	ServerTime  int64  `json:"server_time"`
	ResultCode  string `json:"result_code"`
	EncryptType int    `json:"encrypt_type"`
	Data        []Data `json:"data"`
}
type Data struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PowerSwapIndex struct {
	Id       string `json:"id"`
	Location string `json:"location"`
}

type PowerSwapResp struct {
	RequestID   string        `json:"request_id"`
	ServerTime  int64         `json:"server_time"`
	ResultCode  string        `json:"result_code"`
	EncryptType int           `json:"encrypt_type"`
	Data        PowerSwapInfo `json:"data"`
}
type PowerSwapInfo struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Type                 string   `json:"type"`
	Location             string   `json:"location"`
	OperatorID           string   `json:"operator_id"`
	OperatorName         string   `json:"operator_name"`
	Address              string   `json:"address"`
	Construction         string   `json:"construction"`
	RightDescSummary     string   `json:"right_desc_summary"`
	RightDescDetailArray []string `json:"right_desc_detail_array"`
	Model                string   `json:"model"`
}

func getErr(msg string, err error) {
	if err != nil {
		log.Printf("%v err->%v\n", msg, err)
	}
}

func (task *taskInfo) getPowerMapInfo() {
	powerSwapMapInfo := fmt.Sprintf("https://pe-fe-gateway.nio.com/pe/bff/gateway/powermap/h5/charge-map/v1/configs/dictionary?app_ver=5.2.0&client=pc&container=brower&lang=zh&region=CN&app_id=100119&channel=official&timestamp=%d", time.Now().Unix())
	url := powerSwapMapInfo
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	var response PowerMapInfo
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, v := range response.Data {
		if v.Key == "h5_charge_map_power_swap_resource_cdn_link" {
			task.PowerSwapInfoUrl = v.Value
		}
	}
}

func (task *taskInfo) getPowerSwapList() {
	resp, err := http.Get(task.PowerSwapInfoUrl)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	var newPowerSwapList []PowerSwapIndex
	err = json.NewDecoder(resp.Body).Decode(&newPowerSwapList)
	getErr("get power swap list error", err)
	// task.PowerSwapList = newPowerSwapList[:len(newPowerSwapList)-1]
	if len(task.PowerSwapList) != 0 {
		task.ChangeList = difference(newPowerSwapList, task.PowerSwapList)
	} else {
		task.ChangeList = []PowerSwapIndex{}
	}
	fmt.Println("历史换电站数量", len(task.PowerSwapList), "当前换电站数量", len(newPowerSwapList), "变更数量", len(task.ChangeList))
	task.PowerSwapList = newPowerSwapList
}

func (task *taskInfo) getPowerDetailInfo() {
	for _, v := range task.ChangeList {
		url := fmt.Sprintf("https://pe-fe-gateway.nio.com/pe/bff/gateway/powermap/h5/charge-map/v1/power-swap/detail?app_ver=5.2.0&client=pc&container=brower&lang=zh&region=CN&app_id=100119&channel=official&swap_id=%s&timestamp=%d", v.Id, time.Now().Unix())
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		var response PowerSwapResp
		err = json.NewDecoder(resp.Body).Decode(&response)
		getErr("get power swap list error", err)
		task.sendPowerSwapInfoByDingTalkInfo(response.Data)
	}

}

func (task *taskInfo) sendPowerSwapInfoByDingTalkInfo(info PowerSwapInfo) {
	DingTalk := ding.Webhook{
		AccessToken: task.DingtalkToken,
		Secret:      task.DingtalkSec,
	}
	msg := fmt.Sprintf("[换电站上线] \n名称: %s \n地址: %s\n", info.Name, info.Address)
	result := DingTalk.SendMessageText(msg)
	fmt.Println(result)
}

func difference(a, b []PowerSwapIndex) []PowerSwapIndex {
	m := make(map[PowerSwapIndex]bool)
	diff := []PowerSwapIndex{}

	// 将 b 中的元素添加到 map 中
	for _, item := range b {
		m[item] = true
	}

	// 遍历 a 中的元素，如果不在 map 中，说明 a 中有但 b 中没有
	for _, item := range a {
		if _, ok := m[item]; !ok {
			diff = append(diff, item)
		}
	}

	return diff
}
func main() {
	cfg, err := ini.Load("config.ini")
	getErr("load config", err)

	task := &taskInfo{DingtalkToken: cfg.Section("").Key("DingTalkToken").String(), DingtalkSec: cfg.Section("").Key("DingTalkSec").String()}
	task.sendPowerSwapInfoByDingTalkInfo(PowerSwapInfo{Name: "测试消息", Address: "测试地址"})
	ticker := time.NewTicker(30 * time.Minute) // 创建一个每 30 分钟触发一次的 Ticker
	defer ticker.Stop()                        // 关闭 Ticker
	for {
		select {
		case <-ticker.C: // Ticker 触发的事件
			task.getPowerMapInfo()
			task.getPowerSwapList()
			task.getPowerDetailInfo()
		}
	}
}
