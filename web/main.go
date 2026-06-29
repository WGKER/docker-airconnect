package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type AirUPnP struct {
	XMLName    xml.Name `xml:"airupnp"`
	Common     Common   `xml:"common"`
	MainLog    string   `xml:"main_log"`
	UpnpLog    string   `xml:"upnp_log"`
	UtilLog    string   `xml:"util_log"`
	RaopLog    string   `xml:"raop_log"`
	LogLimit   int      `xml:"log_limit"`
	MaxPlayers int      `xml:"max_players"`
	Binding    string   `xml:"binding"`
	Ports      string   `xml:"ports"`
	Devices    []Device `xml:"device"`
}

type Common struct {
	Enabled    int    `xml:"enabled"`
	MaxVolume  int    `xml:"max_volume"`
	HttpLength int    `xml:"http_length"`
	UpnpMax    int    `xml:"upnp_max"`
	Codec      string `xml:"codec"`
	Metadata   int    `xml:"metadata"`
	Flush      int    `xml:"flush"`
	Artwork    string `xml:"artwork"`
	Latency    string `xml:"latency"`
	Drift      int    `xml:"drift"`
}

type Device struct {
	UDN     string `xml:"udn"`
	Name    string `xml:"name"`
	Mac     string `xml:"mac"`
	Enabled int    `xml:"enabled"`
}

const configPath = "/config/config.xml"

const htmlTemplate = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>AirConnect 设置</title>
    <style>
        * {box-sizing:border-box; margin:0; padding:0; font-family:Arial, sans-serif;}
        body {background:#f5f7fa; padding:20px; max-width:700px; margin:0 auto;}
        .card {background:white; padding:24px; border-radius:16px; margin-bottom:20px; box-shadow:0 2px 12px rgba(0,0,0,0.08);}
        h1 {color:#2d3748; margin-bottom:24px; text-align:center; font-size:22px;}
        h2 {color:#4a5568; margin:20px 0 12px; font-size:16px; border-left:4px solid #3498db; padding-left:10px;}
        .item {display:flex; justify-content:space-between; align-items:center; padding:14px 10px; border-bottom:1px solid #f1f1f1;}
        .name {font-size:15px; color:#2d3748; font-weight:500;}
        .save {width:100%; background:#3498db; color:white; border:none; padding:14px; border-radius:12px; font-size:16px; margin-top:20px; cursor:pointer; font-weight:bold;}
        .save:hover {background:#2980b9;}
        .msg {text-align:center; color:#27ae60; margin:14px 0; font-weight:bold;}
        .toggle {position:relative; width:50px; height:26px;}
        .toggle input {opacity:0; width:0; height:0;}
        .slider {position:absolute; cursor:pointer; top:0; left:0; right:0; bottom:0; background:#ccc; transition:.3s; border-radius:34px;}
        .slider:before {position:absolute; content:""; height:20px; width:20px; left:3px; bottom:3px; background:white; transition:.3s; border-radius:50%;}
        input:checked + .slider {background:#3498db;}
        input:checked + .slider:before {transform:translateX(24px);}
        .version {text-align:center; color:#999; font-size:12px; margin-top:16px;}
    </style>
</head>
<body>
    <div class="card">
        <h1>🔊 AirConnect 设置</h1>
        {{if .Msg}}
        <div class="msg">{{.Msg}}</div>
        {{end}}
        <form method="post">
            <h2>🌍 全局转换</h2>
            <div class="item">
                <span class="name">扫描开关</span>
                <label class="toggle">
                    <input type="checkbox" name="global_enabled" {{if eq .Config.Common.Enabled 1}}checked{{end}}>
                    <span class="slider"></span>
                </label>
            </div>
            <h2>🎵 音箱分控</h2>
            {{range $index, $device := .Config.Devices}}
            <div class="item">
                <span class="name">{{$device.Name}}</span>
                <label class="toggle">
                    <input type="checkbox" name="device_{{$index}}" {{if eq $device.Enabled 1}}checked{{end}}>
                    <span class="slider"></span>
                </label>
            </div>
            {{end}}
            <button class="save" type="submit">💾 保存并重启生效</button>
        </form>
    </div>
    <div class="version">AirConnect 版本：1.10.1</div>
</body>
</html>
`

type PageData struct {
	Config *AirUPnP
	Msg    string
}

func loadConfig() (*AirUPnP, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var config AirUPnP
	err = xml.Unmarshal(data, &config)
	return &config, err
}

func saveConfig(config *AirUPnP) error {
	data, err := xml.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, append([]byte(xml.Header), data...), 0644)
}

// ✅ 官方镜像原生重启命令（100% 适配 /init s6-rc）
func restartAirConnect() {
	time.Sleep(1 * time.Second)
	cmd := exec.Command("s6-rc", "restart", "airupnp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("重启airupnp失败: %v, 输出: %s\n", err, string(output))
	} else {
		fmt.Println("airupnp服务重启成功")
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	config, err := loadConfig()
	if err != nil {
		http.Error(w, "加载配置失败", 500)
		return
	}

	msg := ""
	if r.Method == http.MethodPost {
		if r.PostFormValue("global_enabled") != "" {
			config.Common.Enabled = 1
		} else {
			config.Common.Enabled = 0
		}

		for i := range config.Devices {
			key := fmt.Sprintf("device_%d", i)
			if r.PostFormValue(key) != "" {
				config.Devices[i].Enabled = 1
			} else {
				config.Devices[i].Enabled = 0
			}
		}

		err := saveConfig(config)
		if err != nil {
			msg = "❌ 保存失败"
		} else {
			msg = "✅ 保存成功，服务已自动重启生效"
			go restartAirConnect()
		}
	}

	tpl, _ := template.New("ui").Parse(htmlTemplate)
	tpl.Execute(w, PageData{Config: config, Msg: msg})
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("WebUI 已启动 :8087")
	http.ListenAndServe(":8087", nil)
}
