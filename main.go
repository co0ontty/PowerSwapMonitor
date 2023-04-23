package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/wanghuiyt/ding"
	"gopkg.in/ini.v1"
)

type taskInfo struct {
	DingtalkToken     string
	DingtalkSec       string
	Longitude         string
	Latitude          string
	Distance          string
	PowerSwapInfoUrl  string
	PowerSwapList     []PowerSwapIndex
	PowerSwapNameList []string
	PowerSwapCount    int
	ChangeList        []PowerSwapIndex
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

type PowerMapCountResp struct {
	RequestID   string            `json:"request_id"`
	ServerTime  int64             `json:"server_time"`
	ResultCode  string            `json:"result_code"`
	EncryptType int               `json:"encrypt_type"`
	Data        PowerMapCountInfo `json:"data"`
}
type PowerMapCountInfo struct {
	StatisticUpdateTime          string `json:"statistic_update_time"`
	CurrentAddress               string `json:"current_address"`
	SwapNumber                   string `json:"swap_number"`
	NioNpcChargerNumber          string `json:"nio_npc_charger_number"`
	NioNpcConnectorNumber        string `json:"nio_npc_connector_number"`
	NioDestChargerNumber         string `json:"nio_dest_charger_number"`
	NioDestConnectorNumber       string `json:"nio_dest_connector_number"`
	ThirdConnectorNumber         string `json:"third_connector_number"`
	PsDistrictHousingCoveredRate string `json:"ps_district_housing_covered_rate"`
	HighSpeedSwapNumber          string `json:"high_speed_swap_number"`
	NioChargerNumber             string `json:"nio_charger_number"`
	NioConnectorNumber           string `json:"nio_connector_number"`
	TotalSwapTimesNumber         string `json:"total_swap_times_number"`
}

type PowerMapAroundInfo struct {
	RequestID   string         `json:"request_id"`
	ServerTime  int64          `json:"server_time"`
	ResultCode  string         `json:"result_code"`
	EncryptType int            `json:"encrypt_type"`
	Data        PowerMapAround `json:"data"`
}
type Powers struct {
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
}
type PowerMapAround struct {
	Powers []Powers `json:"powers"`
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
	msg := fmt.Sprintf("[换电站推送] \n名称: %s \n地址: %s\n", info.Name, info.Address)
	result := DingTalk.SendMessageText(msg)
	fmt.Println("DingTalk Response:\n", result)
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
			fmt.Println("debug info", item)
			diff = append(diff, item)
		}
	}

	return diff
}

func checkConfig() {
	if _, err := os.Stat("config.ini"); os.IsNotExist(err) {
		// 文件不存在，创建 config.ini 文件
		file, err := os.Create("config.ini")
		if err != nil {
			fmt.Println("无法创建 config.ini 文件:", err)
			return
		}
		defer file.Close()

		// 写入默认内容到文件
		_, err = file.Write([]byte("DingTalkToken  = 018949c267*****************\nDingTalkSec = SEC700b975e*******************\nLatitude = 39.853147\nLongitude = 116.673329\nDistance = 19600"))
		if err != nil {
			fmt.Println("无法写入到 config.ini 文件:", err)
			return
		}
		fmt.Println("已成功创建 config.ini 文件,请填写相关信息后再次运行")
		os.Exit(0)
	} else {
		fmt.Println("Blue Sky Coming !!!")
	}
}

func (task *taskInfo) getPowerMapCountInfo() {
	url := fmt.Sprintf("https://chargermap-api.nio.com/app/api/pe/h5/charge-map/v1/power/around/summary?app_id=100119&timestamp=%d", time.Now().Unix())
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	var response PowerMapCountResp
	err = json.NewDecoder(resp.Body).Decode(&response)
	getErr("get power swap list error", err)
	newSwapCount, _ := strconv.Atoi(strings.ReplaceAll(response.Data.SwapNumber, ",", ""))
	if newSwapCount != task.PowerSwapCount {
		msg := fmt.Sprintf("[换电站数量变化]：[%d] -> [%d]", task.PowerSwapCount, newSwapCount)
		DingTalk := ding.Webhook{
			AccessToken: task.DingtalkToken,
			Secret:      task.DingtalkSec,
		}
		DingTalk.SendMessageText(msg)
	}
	task.PowerSwapCount = newSwapCount
}

func (task *taskInfo) getPowerInfo() {
	url := fmt.Sprintf("https://chargermap-api.nio.com/app/api/pe/h5/charge-map/v2/power/around?with_national_model=false&latitude=%s&longitude=%s&distance=%s&app_id=100119&timestamp=%d", task.Latitude, task.Longitude, task.Distance, time.Now().Unix())
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	var response PowerMapAroundInfo
	err = json.NewDecoder(resp.Body).Decode(&response)
	getErr("get power swap list error", err)
	newPowerNameList := []string{}
	for _, v := range response.Data.Powers {
		if v.Type != "PowerSwap" {
			continue
		}
		if len(task.PowerSwapNameList) != 0 && !IsContain(task.PowerSwapNameList, v.Name) {
			msg := fmt.Sprintf("[换电站变化] [%s] [%s]", v.Name, v.Address)
			fmt.Println(msg)
			task.sendPowerSwapInfoByDingTalkInfo(PowerSwapInfo{Name: v.Name, Address: v.Address})
		}
		newPowerNameList = append(newPowerNameList, v.Name)
	}
	task.PowerSwapNameList = newPowerNameList
}

func IsContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func main() {
	checkConfig()
	cfg, err := ini.Load("config.ini")
	getErr("load config", err)
	task := &taskInfo{DingtalkToken: cfg.Section("").Key("DingTalkToken").String(), DingtalkSec: cfg.Section("").Key("DingTalkSec").String(), Latitude: "30.25308298", Longitude: "120.2155118", Distance: "19600000"}
	if cfg.Section("").Key("Latitude") != nil {
		task.Latitude = cfg.Section("").Key("Latitude").String()
	}
	if cfg.Section("").Key("Longitude") != nil {
		task.Longitude = cfg.Section("").Key("Longitude").String()
	}
	if cfg.Section("").Key("Distance") != nil {
		task.Distance = cfg.Section("").Key("Distance").String()
	}
	task.getPowerInfo()
	task.sendPowerSwapInfoByDingTalkInfo(PowerSwapInfo{Name: "配置成功", Address: "列表初始化完成"})
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			task.getPowerInfo()
		}
	}
}
