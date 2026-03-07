# axis-node

`axis-node` 是独立的 node 端项目，用于向 Axis 管理端完成节点注册与后续心跳上报。

## 定位

`axis-node` 只做节点代理能力，不承担控制平面职责。

当前阶段包含：

- 节点 UUID 本地持久化
- 节点向 Axis 管理端注册
- 基于 `.env` / 环境变量的最小配置加载
- 周期性上报 CPU / 内存 / 磁盘使用率

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
sudo cp /apps/axis-node/deployments/systemd/axis-node.service /etc/systemd/system/axis-node.service
sudo systemctl daemon-reload
sudo systemctl enable --now axis-node.service
```

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
- `AXIS_NODE_DISK_PATH` 默认 `/`
- `agent` 启动后会先注册，再按配置周期持续上报最新资源指标

## License

MIT
