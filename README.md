# axis-node

`axis-node` 是独立的 node 端项目，用于向 Axis 管理端完成节点注册与后续心跳上报。

## 定位

`axis-node` 只做节点代理能力，不承担控制平面职责。

当前阶段包含：

- 节点 UUID 本地持久化
- 节点向 Axis 管理端注册
- 基于 `.env` / 环境变量的最小配置加载
- 周期性上报 CPU 核数、使用率；内存/Swap 总量与已用；全部磁盘挂载点明细
- 以可选 provider 方式采集本地监控快照，并随节点心跳统一上报 `monitoring_snapshot`
- 启动时自动探测公网 IP 并随上报发送
- 如果 Axis 管理平面启用了可选 DNS 模块，则在首次成功上报公网 IP 后由管理平面自动处理 DNS 记录

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
# 首次部署或后续更新（脚本会先停止 axis-node.service，
# 如果检测到 wt0 网卡，还会自动把 .env 中的
# AXIS_NODE_MANAGEMENT_ADDRESS 更新为 wt0 的 IPv4:端口；
# 如果存在 NetStone 的 server_region_mapping.yaml，
# 还会根据主机名前缀自动同步 AXIS_NODE_REGION / AXIS_NODE_ZONE，
# 再替换二进制与 unit，最后重新启动）
cd /apps/axis-node
./init.sh
```

`axis-node.service` 依赖 `axisd.service`，会在其启动后自动拉起。

`init.sh` 的自动同步规则：

- 如果存在 `wt0` 且能读取到 IPv4，则自动把 `.env` 中的 `AXIS_NODE_MANAGEMENT_ADDRESS` 更新为 `<wt0-ip>:<原端口>`
- 如果原值里没有端口，则默认使用 `9090`
- 如果未检测到 `wt0`，则保留 `.env` 现有值不变
- 如果存在 `NetStone/NetStone/conf/server_region_mapping.yaml`，则按当前主机名第一个 `-` 之前的前缀查询 `prefix_map`，并将 `axis_region` / `country_code` 分别写回 `.env` 的 `AXIS_NODE_REGION` / `AXIS_NODE_ZONE`
- 如果映射文件不存在、主机名前缀未命中或映射字段不完整，则保留 `.env` 现有的 `AXIS_NODE_REGION` / `AXIS_NODE_ZONE`

如果你想手动执行，同样建议先停服务再替换二进制，避免 `/usr/local/bin/axis-node` 被占用：

```bash
cd /apps/axis-node
go build -o axis-node ./cmd/axis-node
sudo systemctl stop axis-node.service
sudo install -m 0755 axis-node /usr/local/bin/axis-node
sudo install -m 0644 deployments/systemd/axis-node.service /etc/systemd/system/axis-node.service
sudo systemctl daemon-reload
sudo systemctl enable --now axis-node.service
```

## 配置项

- `AXIS_NODE_SERVER_URL`
- `AXIS_NODE_MANAGEMENT_ADDRESS`
- `AXIS_NODE_REGION`：大洲（asia、europe、australia、north_america、south_america）
- `AXIS_NODE_ZONE`：可用区，ISO-3166-1 alpha-2 国家代码（如 SG、CN、US），必填
- `AXIS_NODE_HOSTNAME`
- `AXIS_NODE_STATUS`
- `AXIS_NODE_UUID_FILE`
- `AXIS_NODE_REPORT_INTERVAL_SEC`
- `AXIS_NODE_DISK_PATH`
- `AXIS_NODE_SHARED_TOKEN`
- `AXIS_NODE_MONITORING_ENABLED`
- `AXIS_NODE_MONITORING_GO_SIDECAR_ENABLED`
- `AXIS_NODE_SIDECAR_STATS_URL`
- `AXIS_NODE_SIDECAR_STATS_TIMEOUT_SEC`

## 说明

- 如果本地没有 UUID 文件，`axis-node` 会自动生成 `uuid4`
- 生成后的 UUID 默认持久化到 `/data/axis-node/node-uuid`，下次启动继续复用
- Axis 管理端返回的最终 UUID 会再次回写本地文件，确保身份一致
- `axis-node` 会优先读取项目根目录 `.env`
- 通过 systemd 运行时，默认会使用宿主机主机名作为 `AXIS_NODE_HOSTNAME`
- 如果不通过 systemd 运行，则回退到 `os.Hostname()`
- `AXIS_NODE_HOSTNAME` 会直接显示在 Axis 管理端的服务器列表中
- 升级后如果发现旧路径 `./data/node-uuid` 存在且新路径不存在，`axis-node` 会在首次启动时自动迁移到 `/data/axis-node/node-uuid`
- `AXIS_NODE_REPORT_INTERVAL_SEC` 默认 10 秒
- `AXIS_NODE_DISK_PATH` 默认 `/`（仅用于兼容，实际会采集全部挂载点）
- `AXIS_NODE_MONITORING_ENABLED` 默认 `true`
- `AXIS_NODE_MONITORING_GO_SIDECAR_ENABLED` 默认 `true`
- `AXIS_NODE_SIDECAR_STATS_URL` 默认 `http://127.0.0.1:8086/api/v1/internal/workload-stats`
- `AXIS_NODE_SIDECAR_STATS_TIMEOUT_SEC` 默认 3 秒
- `agent` 启动后会先注册，再按配置周期持续上报最新资源指标
- `monitoring_snapshot` 是通用监控 envelope；默认会启用 provider 采集，也可通过 `AXIS_NODE_MONITORING_ENABLED=false` 整体关闭
- `go-sidecar` 本地监控 provider 默认启用，如需关闭可设置 `AXIS_NODE_MONITORING_GO_SIDECAR_ENABLED=false`
- 如果监控 provider 全部关闭，`axis-node` 仍会继续正常注册与心跳上报，只是不附带 `monitoring_snapshot`
- 缺少 `AXIS_NODE_ZONE` 时，`axis-node` 会在启动时直接退出
- 公网 IP 通过外部服务自动探测，探测失败时为空，不阻塞上报
- 磁盘信息按全部挂载点上报，伪文件系统（proc、tmpfs 等）已过滤
- Cloudflare 等 DNS 服务商配置只需要放在 Axis 管理平面；`axis-node` 无需配置相关 Token 或域名参数
- 生产环境需要保证 `/data/axis-node/` 对 `axis-node` 进程可写

## 可选本地监控 Provider

`axis-node` 已经具备通用监控 provider 框架，默认启用 `go-sidecar` 本地监控 provider；如果目标环境没有 `go-sidecar`，可以显式关闭。

启用 `go-sidecar` workload 采集时，至少需要：

```env
AXIS_NODE_MONITORING_ENABLED=true
AXIS_NODE_MONITORING_GO_SIDECAR_ENABLED=true
AXIS_NODE_SIDECAR_STATS_URL=http://127.0.0.1:8086/api/v1/internal/workload-stats
AXIS_NODE_SIDECAR_STATS_TIMEOUT_SEC=3
```

说明：

- `monitoring_snapshot` 是通用 JSON 快照，不绑定某个具体业务
- `go-sidecar` 只是当前内置的一个 provider，后续可以继续扩展其他本地采集源
- provider 采集失败不会中断整次节点心跳，只会在快照中记录该 source 的错误状态
- 当 `axis-node` 跑在宿主机而 `go-sidecar` 跑在 Docker 容器时，`go-sidecar` 需要把宿主机/容器网络来源加入 `WORKLOAD_STATS_TRUSTED_CIDRS`
- 当前 NetStone 默认部署建议至少信任：`127.0.0.1/32,10.10.0.0/16,10.8.0.0/16`

## License

MIT
