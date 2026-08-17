# gRPC 契约

服务间通信契约(对外 REST 见 ../openapi):

- quadlink/v1:扫码绑定/解绑校验(阶段6,师傅端→服务端)
- aaa/v1:停复机联动、授权查询(阶段7,billing→aaa)
- device/v1:OLT 指标上报(阶段7,collector→server)
- provision/v1:配置下发任务(阶段7,order→provisioner)

命名规范:`<domain>/v1/<domain>.proto`,生成:`make proto`
