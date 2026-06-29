package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"time" // 新增延迟所需包
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
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>AirConnect设置</title>
    <style>
        * {box-sizing:border-box; margin:0; padding:0; font-family:Arial, sans-serif;}
        body {
            background:#f5f7fa;
            padding:12px;
            max-width:700px;
            margin:0 auto;
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
            gap:10px;
        }
        .name-text {
            font-size:14px;
            color:#2d3748;
            font-weight:500;
            cursor:pointer;
            flex:1;
        }
        .name-input {
            flex:1;
            padding:4px 6px;
            font-size:14px;
            border:1px solid #3498db;
            border-radius:4px;
            outline:none;
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
                <span class="name-text">扫描开关</span>
                <label class="toggle">
                    <input id="global_enabled" type="checkbox" name="global_enabled" {{if eq .Config.Common.Enabled 1}}checked{{end}}>
                    <span class="slider"></span>
                </label>
            </div>
            <h2>🎵 音箱分控</h2>
            {{range $index, $device := .Config.Devices}}
            <div class="item" data-index="{{$index}}">
                <span class="name-text" data-idx="{{$index}}">{{$device.Name}}</span>
                <input type="hidden" name="device_name_{{$index}}" class="hidden-name" value="{{$device.Name}}">
                <label class="toggle">
                    <input class="device-checkbox" data-index="{{$index}}" type="checkbox" name="device_{{$index}}" {{if eq $device.Enabled 1}}checked{{end}}>
                    <span class="slider"></span>
                </label>
            </div>
            {{end}}
            <button id="submitBtn" class="save" type="submit">💾 保存并重启生效</button>
        </form>
    </div>
    <div class="version">AirConnect 版本：{{.Version}}</div>

<script>
let originState = {
    global: false,
    devices: [],
    names: []
};
const submitBtn = document.getElementById('submitBtn');
window.addEventListener('DOMContentLoaded', function(){
    // 提示3秒自动淡出
    const msgBox = document.querySelector('.msg');
    if(msgBox){
        setTimeout(()=>{
            msgBox.classList.add('fade-out');
            setTimeout(()=>msgBox.remove(), 800);
        }, 3000);
    }

    // 初始化原始开关+名称快照
    const globalInput = document.getElementById('global_enabled');
    originState.global = globalInput.checked;
    const deviceItems = document.querySelectorAll('.item[data-index]');
    deviceItems.forEach(item=>{
        const idx = item.dataset.index;
        const check = item.querySelector('.device-checkbox');
        const nameVal = item.querySelector('.hidden-name').value;
        originState.devices.push(check.checked);
        originState.names.push(nameVal);
    })

    // 点击名称切换编辑框
    document.querySelectorAll('.name-text').forEach(textSpan=>{
        textSpan.addEventListener('click', function(){
            const idx = this.dataset.idx;
            const parent = this.parentElement;
            const hiddenInput = parent.querySelector('.hidden-name');
            const currentVal = hiddenInput.value;
            const input = document.createElement('input');
            input.className = 'name-input';
            input.value = currentVal;
            input.addEventListener('keydown', e=>{
                if(e.key === 'Enter') input.blur();
            })
            input.addEventListener('blur', ()=>{
                hiddenInput.value = input.value.trim() || currentVal;
                const newSpan = document.createElement('span');
                newSpan.className = 'name-text';
                newSpan.dataset.idx = idx;
                newSpan.textContent = hiddenInput.value;
                newSpan.addEventListener('click', ()=>newSpan.click());
                parent.replaceChild(newSpan, input);
            })
            parent.replaceChild(input, this);
            input.focus();
        })
    })

    // 表单提交【修复：AJAX异步提交，不卸载页面，轮询正常执行】
    const form = document.getElementById('configForm');
    form.addEventListener('submit', async function(e){
        e.preventDefault();
        let currentGlobal = document.getElementById('global_enabled').checked;
        let currentDevices = [];
        let currentNames = [];
        document.querySelectorAll('.device-checkbox').forEach(cb=>currentDevices.push(cb.checked));
        document.querySelectorAll('.hidden-name').forEach(h=>currentNames.push(h.value));

        let isChanged = false;
        if(currentGlobal !== originState.global){
            isChanged = true;
        }else{
            for(let i=0; i<originState.devices.length; i++){
                if(currentDevices[i] !== originState.devices[i] || currentNames[i] !== originState.names[i]){
                    isChanged = true;
                    break;
                }
            }
        }

        if(!isChanged){
            alert("未检测到任何配置修改，无需保存。");
            return;
        }
        const confirmSave = confirm("确认保存配置并重启服务？重启后页面会短暂断开。");
        if(!confirmSave) return;

        // 锁定按钮
        submitBtn.disabled = true;
        submitBtn.textContent = "⏳ 保存中，等待服务重启...";

        // 构造表单数据
        const formData = new FormData(form);
        try {
            // 异步POST提交，页面不刷新
            const res = await fetch('/', {
                method: 'POST',
                body: formData,
                signal: AbortSignal.timeout(5000)
            });
            // 提交成功，开始循环检测服务恢复
            function tryReload() {
                fetch('/', {cache:"no-store", signal: AbortSignal.timeout(1500)})
                .then(()=>{
                    // 服务恢复，重载页面读取最新配置
                    location.href = "/";
                })
                .catch(()=>{
                    setTimeout(tryReload, 1500);
                });
            }
            setTimeout(tryReload, 800);
        } catch (err) {
            alert("保存请求异常，请稍后重试");
            submitBtn.disabled = false;
            submitBtn.textContent = "💾 保存并重启生效";
        }
    });
});
</script>
</body>
</html>
`

type PageData struct {
	MsgType string
	Msg     string
	Config  *AirUPnP
	Version string
}

// loadConfig 每次页面GET实时读取磁盘配置，无缓存
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

// 【修复：goroutine异步延迟杀进程，不阻塞HTTP 302响应】
func restartContainer() {
	fmt.Println("触发全容器服务重载，延迟500ms执行重启")
	go func() {
		time.Sleep(500 * time.Millisecond)
		cmd := exec.Command("pkill", "-f", "s6-svscan")
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("重载触发日志: err=%v, output=%s\n", err, string(out))
		}
	}()
}

func handler(w http.ResponseWriter, r *http.Request, pageVer string) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// 每次访问实时读取最新配置文件
	config, err := loadConfig()
	if err != nil {
		http.Error(w, "加载配置失败", 500)
		return
	}

	msg := ""
	msgType := ""
	if r.Method == http.MethodPost {
		// 全局开关
		if r.PostFormValue("global_enabled") != "" {
			config.Common.Enabled = 1
		} else {
			config.Common.Enabled = 0
		}

		// 同步音箱开关+编辑后的名称
		for i := range config.Devices {
			ckKey := fmt.Sprintf("device_%d", i)
			nameKey := fmt.Sprintf("device_name_%d", i)
			if r.PostFormValue(ckKey) != "" {
				config.Devices[i].Enabled = 1
			} else {
				config.Devices[i].Enabled = 0
			}
			newName := r.PostFormValue(nameKey)
			if newName != "" {
				config.Devices[i].Name = newName
			}
		}

		err := saveConfig(config)
		if err != nil {
			msg = "❌ 保存失败"
			msgType = "error"
		} else {
			restartContainer()
			// 提交成功返回200，前端AJAX接管轮询
			return
		}
	}

	tpl, err := template.New("ui").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "模板解析失败: "+err.Error(), 500)
		return
	}
	tpl.Execute(w, PageData{
		Config:  config,
		Msg:     msg,
		MsgType: msgType,
		Version: pageVer,
	})
}

func main() {
	pageVersion := os.Getenv("APP_VERSION")
	if pageVersion == "" {
		pageVersion = "dev"
	}
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
