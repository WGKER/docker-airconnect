package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
	Devices    []Device `xml:"devices>device"`
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

// 新增Mac字段用于前端精准匹配
type DevStatus struct {
	UDN     string `json:"udn"`
	Name    string `json:"name"`
	Mac     string `json:"mac"`
	Playing bool   `json:"playing"`
}

const configPath = "/config/config.xml"

const htmlTemplate = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>AirConnect设置面板</title>
    <style>
        * {box-sizing:border-box; margin:0; padding:0; font-family:system-ui, sans-serif;}
        body {
            background:#f5f7fa;
            padding:12px;
            max-width:700px;
            margin:0 auto;
            font-size:14px;
        }
        .card {
            background:#fff;
            padding:16px;
            border-radius:12px;
            margin-bottom:16px;
            box-shadow:0 2px 12px rgba(0,0,0,0.06);
        }
        h1 {
            color:#1f2937;
            margin-bottom:20px;
            text-align:center;
            font-size:20px;
        }
        h2 {
            color:#374151;
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
            color:#1f2937;
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
        .status-text {
            font-size:12px;
            padding:2px 6px;
            border-radius:6px;
            flex-shrink:0;
        }
        .playing {
            background:#dcfce7;
            color:#16a34a;
        }
        .idle {
            background:#f3f4f6;
            color:#6b7280;
        }
        .save {
            width:100%;
            background:#3498db;
            color:#fff;
            border:none;
            padding:14px;
            border-radius:10px;
            font-size:16px;
            margin-top:16px;
            cursor:pointer;
            font-weight:500;
            min-height:48px;
        }
        .save:hover {background:#2980b9;}
        .save:disabled {background:#94b8d8; cursor:not-allowed;}
        .msg {
            text-align:center;
            margin:12px 0;
            font-weight:500;
            transition:opacity 0.8s ease;
            font-size:14px;
        }
        .msg.success {color:#16a34a;}
        .msg.error {color:#dc2626;}
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
            background:#fff;
            transition:.3s;
            border-radius:50%;
        }
        input:checked + .slider {background:#3498db;}
        input:checked + .slider:before {transform:translateX(22px);}
        .version {
            text-align:center;
            color:#9ca3af;
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
                <span class="name-text">扫描启用</span>
                <label class="toggle">
                    <input id="global_enabled" type="checkbox" name="global_enabled" {{if eq .Config.Common.Enabled 1}}checked{{end}}>
                    <span class="slider"></span>
                </label>
            </div>
            <h2>🎵 音箱分控</h2>
            <div id="devListWrap">
            {{range $index, $device := .Config.Devices}}
            <div class="item" data-index="{{$index}}" data-mac="{{$device.Mac}}">
                <span class="name-text" data-idx="{{$index}}">{{$device.Name}}</span>
                <input type="hidden" name="device_name_{{$index}}" class="hidden-name" value="{{$device.Name}}">
                <span class="status-text idle">空闲</span>
                <label class="toggle">
                    <input class="device-checkbox" data-index="{{$index}}" type="checkbox" name="device_{{$index}}" {{if eq $device.Enabled 1}}checked{{end}}>
                    <span class="slider"></span>
                </label>
            </div>
            {{end}}
            </div>
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
const formWrap = document.getElementById('configForm');
let statusTimer = null;

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
    const deviceItems = document.querySelectorAll('.item[data-mac]');
    deviceItems.forEach(item=>{
        const idx = item.dataset.index;
        const check = item.querySelector('.device-checkbox');
        const nameVal = item.querySelector('.hidden-name').value;
        originState.devices.push(check.checked);
        originState.names.push(nameVal);
    })

    // 事件委托：无限次点击修改音箱名称
    formWrap.addEventListener('click', function(e){
        const textSpan = e.target.closest('.name-text');
        if(!textSpan) return;
        const idx = textSpan.dataset.idx;
        const parent = textSpan.parentElement;
        const hiddenInput = parent.querySelector('.hidden-name');
        const currentVal = hiddenInput.value;
        const input = document.createElement('input');
        input.className = 'name-input';
        input.value = currentVal;
        input.addEventListener('keydown', ev=>{
            if(ev.key === 'Enter') input.blur();
        })
        input.addEventListener('blur', ()=>{
            hiddenInput.value = input.value.trim() || currentVal;
            const newSpan = document.createElement('span');
            newSpan.className = 'name-text';
            newSpan.dataset.idx = idx;
            newSpan.textContent = hiddenInput.value;
            parent.replaceChild(newSpan, input);
        })
        parent.replaceChild(input, textSpan);
        input.focus();
    })

    // 修复：纯MAC匹配，不再对比名称，改名不影响状态识别
    function refreshDeviceStatus() {
        fetch('/status', {cache:"no-store", signal: AbortSignal.timeout(2000)})
        .then(res=>res.json())
        .then(statusList=>{
            // 构建mac映射表，一次遍历
            const macMap = {};
            statusList.forEach(d=>{
                macMap[d.Mac] = d.Playing;
            })
            // 遍历页面设备匹配
            document.querySelectorAll('.item[data-mac]').forEach(item=>{
                const mac = item.dataset.mac;
                const statusSpan = item.querySelector('.status-text');
                const playing = macMap[mac] || false;
                if(playing){
                    statusSpan.className = "status-text playing";
                    statusSpan.textContent = "播放";
                }else{
                    statusSpan.className = "status-text idle";
                    statusSpan.textContent = "空闲";
                }
            })
        })
        .catch(()=>{});
    }
    statusTimer = setInterval(refreshDeviceStatus, 2000);

    // 异步提交表单
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
            alert("未检测到配置修改，无需保存。");
            return;
        }
        const confirmSave = confirm("确认保存并重启服务？页面会短暂断开。");
        if(!confirmSave) return;

        submitBtn.disabled = true;
        submitBtn.textContent = "⏳ 保存中，等待重启...";
        clearInterval(statusTimer); // 停止状态轮询

        const formData = new FormData(form);
        try {
            await fetch('/', {
                method: 'POST',
                body: formData,
                signal: AbortSignal.timeout(5000)
            });
            // 循环检测服务恢复
            function tryReload() {
                fetch('/', {cache:"no-store", signal: AbortSignal.timeout(1500)})
                .then(()=>location.href = "/")
                .catch(()=>setTimeout(tryReload,1500));
            }
            setTimeout(tryReload, 800);
        } catch (err) {
            alert("保存请求异常，请重试。");
            submitBtn.disabled = false;
            submitBtn.textContent = "💾 保存并重启生效";
            statusTimer = setInterval(refreshDeviceStatus, 2000);
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

// loadConfig 每次页面访问实时读取磁盘配置
func loadConfig() (*AirUPnP, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg AirUPnP
	err = xml.Unmarshal(data, &cfg)
	return &cfg, err
}

func saveConfig(cfg *AirUPnP) error {
	data, err := xml.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, append([]byte(xml.Header), data...), 0644)
}

// 异步延迟重启s6，不阻塞http响应
func restartContainer() {
	fmt.Println("延迟500ms重载s6服务")
	go func() {
		time.Sleep(500 * time.Millisecond)
		cmd := exec.Command("pkill", "-f", "s6-svscan")
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("重启错误: %v, output: %s\n", err, string(out))
		}
	}()
}

// /status 接口：自动解析日志mac→句柄，返回携带Mac用于前端匹配
func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	conf, err := loadConfig()
	if err != nil {
		_ = json.NewEncoder(w).Encode([]DevStatus{})
		return
	}

	var fullLog string
	// 获取airupnp-docker进程控制台输出
	pidCmd := exec.Command("pgrep", "-f", "airupnp-docker")
	pidOut, pidErr := pidCmd.CombinedOutput()
	if pidErr == nil && len(pidOut) > 0 {
		pid := strings.TrimSpace(string(pidOut))
		tailCmd := exec.Command("tail", "-n", "500", "/proc/"+pid+"/fd/1")
		tailOut, tailErr := tailCmd.CombinedOutput()
		if tailErr == nil {
			fullLog = string(tailOut)
		}
	}

	// 自动解析AddMRDevice行，建立 mac => 设备句柄 映射
	macToHandle := make(map[string]string)
	lines := strings.Split(fullLog, "\n")
	for _, line := range lines {
		if strings.Contains(line, "AddMRDevice") && strings.Contains(line, "with mac") {
			// 提取句柄 [0xffffxxxx]
			handleStart := strings.Index(line, "[0x")
			if handleStart == -1 {
				continue
			}
			handleEnd := strings.Index(line[handleStart:], "]") + handleStart
			handle := line[handleStart : handleEnd+1]
			// 提取mac地址
			macMarker := "with mac "
			macPos := strings.LastIndex(line, macMarker)
			if macPos == -1 {
				continue
			}
			mac := strings.TrimSpace(line[macPos+len(macMarker):])
			macToHandle[mac] = handle
		}
	}

	var result []DevStatus
	for _, dev := range conf.Devices {
		isPlay := false
		mac := dev.Mac
		handle, exist := macToHandle[mac]
		if fullLog != "" && exist && strings.Contains(fullLog, handle) {
			// 播放标记
			hasPlay := strings.Contains(fullLog, "uPNP playing") ||
				strings.Contains(fullLog, "received RECORD") ||
				strings.Contains(fullLog, "received metadata") ||
				strings.Contains(fullLog, "received JPEG image")
			// 停止标记
			hasStop := strings.Contains(fullLog, "TEARDOWN") ||
				strings.Contains(fullLog, "uPNP stopped") ||
				strings.Contains(fullLog, "uPNP stop") ||
				strings.Contains(fullLog, "Stop")
			if hasPlay && !hasStop {
				isPlay = true
			}
		}
		result = append(result, DevStatus{
			UDN:     dev.UDN,
			Name:    dev.Name,
			Mac:     dev.Mac, // 新增MAC返回前端用于匹配
			Playing: isPlay,
		})
	}
	_ = json.NewEncoder(w).Encode(result)
}

// 主页面板渲染
func pageHandler(w http.ResponseWriter, r *http.Request, pageVer string) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	conf, err := loadConfig()
	if err != nil {
		http.Error(w, "读取配置文件失败", 500)
		return
	}

	msg := ""
	msgType := ""
	if r.Method == http.MethodPost {
		// 全局开关
		if r.PostFormValue("global_enabled") != "" {
			conf.Common.Enabled = 1
		} else {
			conf.Common.Enabled = 0
		}
		// 遍历设备同步开关与自定义名称
		for i := range conf.Devices {
			ckKey := fmt.Sprintf("device_%d", i)
			nameKey := fmt.Sprintf("device_name_%d", i)
			if r.PostFormValue(ckKey) != "" {
				conf.Devices[i].Enabled = 1
			} else {
				conf.Devices[i].Enabled = 0
			}
			newName := r.PostFormValue(nameKey)
			if newName != "" {
				conf.Devices[i].Name = newName
			}
		}
		errSave := saveConfig(conf)
		if errSave != nil {
			msg = "❌ 保存失败"
			msgType = "error"
		} else {
			restartContainer()
			return
		}
	}

	tpl, err := template.New("ui").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "模板解析失败: "+err.Error(), 500)
		return
	}
	_ = tpl.Execute(w, PageData{
		MsgType: msgType,
		Msg:     msg,
		Config:  conf,
		Version: pageVer,
	})
}

func main() {
	pageVersion := os.Getenv("APP_VERSION")
	if pageVersion == "" {
		pageVersion = "dev"
	}
	// 注册路由
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		pageHandler(w, r, pageVersion)
	})
	fmt.Printf("Web面板启动 :8087 | 版本=%s\n", pageVersion)
	err := http.ListenAndServe(":8087", nil)
	if err != nil {
		fmt.Printf("服务启动异常: %v\n", err)
		os.Exit(1)
	}
}
