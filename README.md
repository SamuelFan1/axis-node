# axis-node

`axis-node` 是独立的 node 端项目，用于向 Axis 管理端完成节点注册与后续心跳上报。

## 定位

`axis-node` 只做节点代理能力，不承担控制平面职责。

当前阶段包含：

- 节点 UUID 本地持久化
- 节点向 Axis 管理端注册
- 基于 `.env` / 环境变量的最小配置加载
- 周期性上报 CPU 核数、使用率；内存/Swap 总量与已用；全部磁盘挂载点明细
- 启动时自动探测公网 IP 并随上报发送

## 命令

注册当前节点：

```bash
cp .env.example .env
./axis-node register
```

启动长期运行的 node agent：

```bash
cp .env.example .env
./axis-node agent
```

推荐生产环境使用 systemd：

```bash
# 构建并部署（需先启动 axisd）
cd /apps/axis-node
go build -o axis-node ./cmd/axis-node
sudo cp axis-node /usr/local/bin/

# 安装 systemd 单元并启动
sudo cp deployments/systemd/axis-node.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now axis-node.service
```

`axis-node.service` 依赖 `axisd.service`，会在其启动后自动拉起。

## 配置项

- `AXIS_NODE_SERVER_URL`
- `AXIS_NODE_MANAGEMENT_ADDRESS`
- `AXIS_NODE_REGION`
- `AXIS_NODE_HOSTNAME`
- `AXIS_NODE_STATUS`
- `AXIS_NODE_UUID_FILE`
- `AXIS_NODE_REPORT_INTERVAL_SEC`
- `AXIS_NODE_DISK_PATH`
- `AXIS_NODE_SHARED_TOKEN`

## 说明

- 如果本地没有 UUID 文件，`axis-node` 会自动生成 `uuid4`
- 生成后的 UUID 会持久化到本地文件，下次启动继续复用
- Axis 管理端返回的最终 UUID 会再次回写本地文件，确保身份一致
- `axis-node` 会优先读取项目根目录 `.env`
- 通过 systemd 运行时，默认会使用宿主机主机名作为 `AXIS_NODE_HOSTNAME`
- 如果不通过 systemd 运行，则回退到 `os.Hostname()`
- `AXIS_NODE_HOSTNAME` 会直接显示在 Axis 管理端的服务器列表中
- `AXIS_NODE_REPORT_INTERVAL_SEC` 默认 10 秒
- `AXIS_NODE_DISK_PATH` 默认 `/`（仅用于兼容，实际会采集全部挂载点）
- `agent` 启动后会先注册，再按配置周期持续上报最新资源指标
- 公网 IP 通过外部服务自动探测，探测失败时为空，不阻塞上报
- 磁盘信息按全部挂载点上报，伪文件系统（proc、tmpfs 等）已过滤

## License

MIT
