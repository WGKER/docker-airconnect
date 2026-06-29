package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
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
    <!-- 手机适配核心视口标签 -->
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>AirConnect设置</title>
    <style>
        * {box-sizing:border-box; margin:0; padding:0; font-family:Arial, sans-serif;}
        body {
            background:#f5f7fa;
            padding:12px;
            max-width:700px;
            margin:0 auto;
            /* 手机基础字体优化 */
            font-size:14px;
        }
        .card {
            background:white;
            padding:16px;
            border-radius:12px;
            margin-bottom:16px;
            box-shadow:0 2px 12px rgba(0,0,0,0.08);
        }
        h1 {
            color:#2d3748;
            margin-bottom:20px;
            text-align:center;
            font-size:20px;
        }
        h2 {
            color:#4a5568;
            margin:16px 0 10px;
            font-size:15px;
            border-left:4px solid #3498db;
            padding-left:8px;
        }
        .item {
            display:flex;
            justify-content:space-between;
            align-items:center;
            padding:12px 6px;
            border-bottom:1px solid #f1f1f1;
        }
        .name {
            font-size:14px;
            color:#2d3748;
            font-weight:500;
        }
        .save {
            width:100%;
            background:#3498db;
            color:white;
            border:none;
            padding:14px;
            border-radius:10px;
            font-size:16px;
            margin-top:16px;
            cursor:pointer;
            font-weight:bold;
            /* 手机增大点击区域 */
            min-height:48px;
        }
        .save:hover {background:#2980b9;}
        .save:disabled {background:#94b8d8; cursor:not-allowed;}
        .msg {
            text-align:center;
            margin:12px 0;
            font-weight:bold;
            transition: opacity 0.8s ease;
            font-size:14px;
        }
        .msg.success {color:#27ae60;}
        .msg.error {color:#e74c3c;}
        .msg.fade-out {opacity:0;}
        .toggle {
            position:relative;
            width:46px;
            height:24px;
            /* 手机开关缩小适配 */
            flex-shrink:0;
        }
        .toggle input {opacity:0; width:0; height:0;}
        .slider {
            position:absolute;
            cursor:pointer;
            top:0; left:0; right:0; bottom:0;
            background:#ccc;
            transition:.3s;
            border-radius:34px;
        }
        .slider:before {
            position:absolute;
            content:"";
            height:18px;
            width:18px;
            left:3px;
            bottom:3px;
            background:white;
            transition:.3s;
            border-radius:50%;
        }
        input:checked + .slider {background:#3498db;}
        input:checked + .slider:before {transform:translateX(22px);}
        .version {
            text-align:center;
            color:#999;
            font-size:11px;
            margin-top:14px;
        }
        /* 小屏幕手机额外压缩 */
        @media (max-width:480px) {
            body {padding:8px;}
            .card {padding:14px; border-radius:10px;}
            h1 {font-size:18px;}
            .item {padding:10px 4px;}
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>🔊 AirConnect 设置</h1>
        {{if .Msg}}
        <div class="msg {{.MsgType}}">{{.Msg}}</div>
        {{end}}
        <form id="configForm" method="post">
            <h2>🌍 全局转换</h2>
            <div class="item">
                <span class="name">扫描开关</span>
                <label class="toggle">
                    <input id="global_enabled" type="checkbox" name="global_enabled" {{if eq .Config.Common.Enabled 1}}checked{{end}}>
                    <span class="slider"></span>
                </label>
            </div>
            <h2>🎵 音箱分控</h2>
            {{range $index, $device := .Config.Devices}}
            <div class="item">
                <span class="name">{{$device.Name}}</span>
                <label class="toggle">
                    <input class="device-checkbox" data-index="{{$index}}" type="checkbox" name="device_{{$index}}" {{if eq $device.Enabled 1}}checked{{end}}>
                    <span class="slider"></span>
                </label>
            </div>
            {{end}}
            <button class="save" type="submit">💾 保存并重启生效</button>
        </form>
    </div>
    <div class="version">AirConnect 版本：{{.Version}}</div>

<script>
// 页面加载完成后记录原始状态快照
let originState = {
    global: false,
    devices: []
};
window.addEventListener('DOMContentLoaded', function(){
    // 提示文字3秒自动淡出消失
    const msgBox = document.querySelector('.msg');
    if(msgBox){
        setTimeout(()=>{
            msgBox.classList.add('fade-out');
            setTimeout(()=>msgBox.remove(), 800);
        }, 3000);
    }

    // 记录全局开关初始状态
    const globalInput = document.getElementById('global_enabled');
    originState.global = globalInput.checked;

    // 记录所有设备勾选初始状态
    const deviceInputs = document.querySelectorAll('.device-checkbox');
    deviceInputs.forEach(input => {
        originState.devices.push(input.checked);
    });

    // 绑定表单提交拦截事件
    const form = document.getElementById('configForm');
    form.addEventListener('submit', function(e){
        e.preventDefault(); // 先阻止原生提交

        // 获取当前最新状态
        let currentGlobal = document.getElementById('global_enabled').checked;
        let currentDevices = [];
        document.querySelectorAll('.device-checkbox').forEach(input => {
            currentDevices.push(input.checked);
        });

        // 对比是否存在修改
        let isChanged = false;
        if(currentGlobal !== originState.global){
            isChanged = true;
        }else{
            for(let i=0; i<originState.devices.length; i++){
                if(currentDevices[i] !== originState.devices[i]){
                    isChanged = true;
                    break;
                }
            }
        }

        // 无修改：提示并退出
        if(!isChanged){
            alert("未检测到任何配置修改，无需保存。");
            return;
        }

        // 有修改：弹出确认框
        const confirmSave = confirm("确认保存配置并重启服务？重启后页面会短暂断开。");
        if(confirmSave){
            form.submit(); // 用户确认，执行提交
        }
    });

    // 容器重启自动刷新逻辑：每2秒检测页面连通性，断开则重载
    function autoReloadOnRestart() {
        fetch('/', {cache:"no-store"})
            .catch(()=>{
                // 请求失败=服务重启中，等待1秒刷新页面
                setTimeout(()=>location.reload(),1000);
                return;
            });
    }
    // 每2秒执行一次检测
    setInterval(autoReloadOnRestart, 2000);
});
</script>
</body>
</html>
`

// PageData 新增Version字段，用于渲染流水线传入的版本号
type PageData struct {
	MsgType string // success / error
	Msg     string
	Config  *AirUPnP
	Version string
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

// 杀死 s6-svscan 主进程强制全容器重载
func restartContainer() {
	fmt.Println("触发全容器服务重载")
	cmd := exec.Command("pkill", "-f", "s6-svscan")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("重载触发日志: err=%v, output=%s\n", err, string(out))
	}
}

func handler(w http.ResponseWriter, r *http.Request, pageVer string) {
	// 禁止浏览器缓存POST表单，杜绝重复提交弹窗缓存
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	config, err := loadConfig()
	if err != nil {
		http.Error(w, "加载配置失败", 500)
		return
	}

	msg := ""
	msgType := ""
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
			// 保存失败：保留页面，展示错误提示
			msg = "❌ 保存失败"
			msgType = "error"
		} else {
			// 保存成功：重启容器 + 302重定向GET首页，清空POST历史
			restartContainer()
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	}

	// 捕获模板解析错误，不再忽略
	tpl, err := template.New("ui").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "模板解析失败: "+err.Error(), 500)
		return
	}
	// 把读取到的版本传入模板渲染
	tpl.Execute(w, PageData{
		Config:  config,
		Msg:     msg,
		MsgType: msgType,
		Version: pageVer,
	})
}

func main() {
	// 从容器环境变量读取流水线传入的版本 APP_VERSION
	pageVersion := os.Getenv("APP_VERSION")
	// 兜底值，本地未传参时显示dev
	if pageVersion == "" {
		pageVersion = "dev"
	}

	// 闭包传递版本参数给handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, pageVersion)
	})

	fmt.Printf("WebUI 已启动 :8087 | 当前版本: %s\n", pageVersion)
	err := http.ListenAndServe(":8087", nil)
	if err != nil {
		fmt.Printf("Web服务启动异常: %v\n", err)
		os.Exit(1)
	}
}
