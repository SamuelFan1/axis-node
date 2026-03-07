# axis-node

`axis-node` 是独立的 node 端项目，用于向 Axis 管理端完成节点注册与后续心跳上报。

## 定位

`axis-node` 只做节点代理能力，不承担控制平面职责。

当前阶段包含：

- 节点 UUID 本地持久化
- 节点向 Axis 管理端注册
- 基于环境变量的最小配置加载

## 命令

注册当前节点：

```bash
AXIS_NODE_SERVER_URL=http://127.0.0.1:9090 \
AXIS_NODE_MANAGEMENT_ADDRESS=10.8.1.11:9090 \
AXIS_NODE_REGION=sgp \
go run ./cmd/axis-node register
```

## 配置项

- `AXIS_NODE_SERVER_URL`
- `AXIS_NODE_MANAGEMENT_ADDRESS`
- `AXIS_NODE_REGION`
- `AXIS_NODE_HOSTNAME`
- `AXIS_NODE_STATUS`
- `AXIS_NODE_UUID_FILE`

## 说明

- 如果本地没有 UUID 文件，`axis-node` 会自动生成 `uuid4`
- 生成后的 UUID 会持久化到本地文件，下次启动继续复用
- Axis 管理端返回的最终 UUID 会再次回写本地文件，确保身份一致

## License

MIT
