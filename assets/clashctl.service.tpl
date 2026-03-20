[Unit]
Description=Mihomo Proxy Service (managed by clashctl)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{.Binary}} -d {{.ConfigDir}}
Restart=always
RestartSec=3
LimitNOFILE=65535
{{if .User}}User={{.User}}
{{end}}{{if .Group}}Group={{.Group}}
{{end}}[Install]
WantedBy=multi-user.target
